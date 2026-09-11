package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"time"

	spark "github.com/breez/breez-sdk-spark-go/breez_sdk_spark"
)

const (
	operatorKindDeposit  = "deposit"
	operatorKindWithdraw = "withdraw"
)

// errLedgerNotBalanced gates Spark payouts when the wallet cannot cover the books.
var errLedgerNotBalanced = errors.New("wallet does not cover the books (imbalance < 0)")

// ledgerStatement is the single source of the wallet-vs-books identity. /admin/ledger
// and the payout gates (redeem, refund worker, admin withdraw) all read it so the
// "enough funds" check is never a second formula.
func (srv *Server) ledgerStatement() (LedgerStatement, error) {
	infoResp, err := srv.ln.GetInfo(spark.GetInfoRequest{})
	if err := sdkErr(err); err != nil {
		return LedgerStatement{}, fmt.Errorf("get sdk info: %w", err)
	}

	stmt, err := srv.getLedgerStats()
	if err != nil {
		return LedgerStatement{}, err
	}

	stmt.finalize(int64(infoResp.BalanceSats) * 1000)
	return stmt, nil
}

// payoutsPaused returns non-nil when the wallet is short of the books. While it
// returns non-nil, redeem / refund pays / admin withdraw must not move Spark sats.
func (srv *Server) payoutsPaused() error {
	stmt, err := srv.ledgerStatement()
	if err != nil {
		return fmt.Errorf("ledger statement: %w", err)
	}
	if !stmt.Balanced {
		return fmt.Errorf("%w: imbalance_msat=%d", errLedgerNotBalanced, stmt.ImbalanceMsat)
	}
	return nil
}

func (srv *Server) handleLedger(w http.ResponseWriter, r *http.Request) {
	stmt, err := srv.ledgerStatement()
	if err != nil {
		slog.Error("ledger statement", "err", err)
		lnurlError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// BalanceSats covers only spendable Spark leaves. Surface what it excludes so
	// the ledger can be reconciled against another wallet showing the same mnemonic:
	// unclaimed on-chain deposits and any token balances are not part of wallet_msat.
	infoResp, err := srv.ln.GetInfo(spark.GetInfoRequest{})
	if err := sdkErr(err); err == nil {
		stmt.TokenBalances = tokenBalancesToJSON(infoResp.TokenBalances)
	}
	stmt.UnclaimedDepositsMsat = srv.unclaimedDepositsMsat()

	writeJSON(w, http.StatusOK, stmt)
}

// unclaimedDepositsMsat returns the total msat held in on-chain deposits the SDK has
// not (yet) claimed into the Spark tree. Returns nil when the query fails — the field
// is then omitted from the ledger response rather than reported as zero.
func (srv *Server) unclaimedDepositsMsat() *int64 {
	resp, rawErr := srv.ln.ListUnclaimedDeposits(spark.ListUnclaimedDepositsRequest{})
	if err := sdkErr(rawErr); err != nil {
		slog.Error("list unclaimed deposits", "err", err)
		return nil
	}
	var total int64
	for _, d := range resp.Deposits {
		total += int64(d.AmountSats) * 1000
	}
	return &total
}

// tokenBalancesToJSON renders SDK token balances as decimal strings (u128 does not
// survive JSON round-trips through float64). Nil when there are no tokens.
func tokenBalancesToJSON(balances map[string]spark.TokenBalance) map[string]string {
	if len(balances) == 0 {
		return nil
	}
	out := make(map[string]string, len(balances))
	for id, tb := range balances {
		if tb.Balance != nil {
			out[id] = tb.Balance.String()
		}
	}
	return out
}

// handleAdminDeposit creates a bolt11 invoice the operator pays from a DIFFERENT
// wallet to top up a deficit. Default amount is the current deficit (sat-rounded).
// The deposit is not booked as equity — it only raises wallet_msat so imbalance
// returns to >= 0.
func (srv *Server) handleAdminDeposit(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AmountMsat int64 `json:"amount_msat"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		lnurlError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	stmt, err := srv.ledgerStatement()
	if err != nil {
		slog.Error("ledger statement", "err", err)
		lnurlError(w, http.StatusInternalServerError, "internal error")
		return
	}
	deficitMsat := int64(0)
	if stmt.ImbalanceMsat < 0 {
		deficitMsat = -stmt.ImbalanceMsat
	}

	amountMsat := req.AmountMsat
	if amountMsat == 0 {
		amountMsat = satRound(deficitMsat)
	}
	if amountMsat < 1000 {
		lnurlError(w, http.StatusBadRequest, "amount_msat must be at least 1000 (1 sat)")
		return
	}
	if amountMsat%1000 != 0 {
		lnurlError(w, http.StatusBadRequest, "amount_msat must be a whole number of sats")
		return
	}

	tx := &FundTx{Msat: amountMsat}
	if err := srv.getCallbackBolt11(tx, "Operator deposit"); err != nil {
		slog.Error("create deposit invoice", "err", err)
		lnurlError(w, http.StatusInternalServerError, "failed to create invoice")
		return
	}

	id, err := srv.insertOperatorTx(operatorKindDeposit, tx.Msat, tx.FeeMsat, tx.PR, "")
	if err != nil {
		slog.Error("insert operator deposit", "err", err)
		lnurlError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":           id,
		"pr":           tx.PR,
		"amount_msat":  tx.Msat,
		"deficit_msat": deficitMsat,
	})
}

// handleAdminWithdraw pays booked equity (fees + dust not yet withdrawn) to a bolt11
// invoice, LNURL-pay, or lightning address. The withdrawable cap is a GROSS budget
// (amount + routing fee). A blank amount means "withdraw everything": the estimated
// routing fee is subtracted from the cap so the send fits inside it, leaving a little
// equity behind if the actual fee comes in higher.
func (srv *Server) handleAdminWithdraw(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Destination string `json:"destination"`
		AmountMsat  int64  `json:"amount_msat"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		lnurlError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Destination == "" {
		lnurlError(w, http.StatusBadRequest, "destination is required")
		return
	}

	if err := srv.payoutsPaused(); err != nil {
		lnurlError(w, http.StatusConflict, err.Error())
		return
	}

	if err := srv.payAdminWithdraw(req.AmountMsat, req.Destination, w); err != nil {
		// payAdminWithdraw already wrote the response for handled cases.
		if !errors.Is(err, errWithdrawHandled) {
			slog.Error("admin withdraw", "err", err)
			lnurlError(w, http.StatusBadGateway, err.Error())
		}
		return
	}
}

// errWithdrawHandled marks errors whose HTTP response was already written.
var errWithdrawHandled = errors.New("withdraw response already written")

// withdrawCapMsat is the gross budget for an admin withdraw (amount + routing fee):
// booked equity still on the books, further limited so voucher/refund/redeem holds
// stay covered by the wallet.
func withdrawCapMsat(s LedgerStatement) int64 {
	liabilities := s.Liabilities.Vouchers.Active.BalanceMsat +
		s.Liabilities.Vouchers.Inactive.BalanceMsat +
		s.Liabilities.Refunds.HeldMsat +
		s.Liabilities.RedeemsPending.HeldMsat
	walletHeadroom := s.WalletMsat - liabilities
	capMsat := s.Equity.AvailableMsat
	if walletHeadroom < capMsat {
		capMsat = walletHeadroom
	}
	if capMsat < 0 {
		capMsat = 0
	}
	return capMsat
}

// satRound rounds down to whole sats.
func satRound(msat int64) int64 {
	if msat < 0 {
		return 0
	}
	return msat / 1000 * 1000
}

// payAdminWithdraw sends to destination (bolt11, LNURL-pay, or lightning address)
// and records the operator_tx with an idempotency key. requestedMsat == 0 means
// "withdraw the full available equity": the routing fee is estimated first and
// subtracted from the cap so the send fits. The confirmed gross (amount + actual
// routing fee) is what reduces equity.available_msat.
func (srv *Server) payAdminWithdraw(requestedMsat int64, destination string, w http.ResponseWriter) error {
	srv.paymentSema.acquireForWithdrawal()
	defer srv.paymentSema.releaseAfter(srv.cfg.paymentCooldown)

	inputType, rawErr := srv.ln.Parse(destination)
	if err := sdkErr(rawErr); err != nil {
		return fmt.Errorf("parse destination: %w", err)
	}

	stmt, err := srv.ledgerStatement()
	if err != nil {
		return fmt.Errorf("ledger statement: %w", err)
	}
	capMsat := withdrawCapMsat(stmt)
	if capMsat < 1000 {
		lnurlError(w, http.StatusConflict, "no equity available to withdraw")
		return errWithdrawHandled
	}

	switch v := inputType.(type) {
	case spark.InputTypeBolt11Invoice:
		return srv.sendAdminWithdrawBolt11(w, requestedMsat, capMsat, v)
	case spark.InputTypeLightningAddress:
		return srv.sendAdminWithdrawLnurl(w, requestedMsat, capMsat, v.Field0.PayRequest)
	case spark.InputTypeLnurlPay:
		return srv.sendAdminWithdrawLnurl(w, requestedMsat, capMsat, v.Field0)
	default:
		return fmt.Errorf("unsupported destination type: %T", inputType)
	}
}

// resolveWithdrawAmount works out the net send amount given the gross cap and the
// estimated routing fee. A blank (0) amount means "full equity": net = cap - fee.
// An explicit amount must fit with the fee inside the cap.
func resolveWithdrawAmount(requestedMsat, capMsat, estimateFeeMsat int64) (int64, error) {
	amountMsat := requestedMsat
	if amountMsat == 0 {
		amountMsat = capMsat - estimateFeeMsat
	}
	amountMsat = satRound(amountMsat)
	if amountMsat < 1000 {
		return 0, fmt.Errorf("available equity %d msat does not cover the routing fee %d msat", capMsat, estimateFeeMsat)
	}
	if amountMsat+estimateFeeMsat > capMsat {
		return 0, fmt.Errorf("amount + routing fee %d msat exceeds withdrawable equity %d msat", amountMsat+estimateFeeMsat, capMsat)
	}
	return amountMsat, nil
}

func (srv *Server) sendAdminWithdrawBolt11(w http.ResponseWriter, requestedMsat, capMsat int64, invoice spark.InputTypeBolt11Invoice) error {
	prepResp, rawPrepErr := srv.ln.PrepareSendPayment(spark.PrepareSendPaymentRequest{
		PaymentRequest: spark.PaymentRequestInput{Input: invoice.Field0.Invoice.Bolt11},
	})
	if err := sdkErr(rawPrepErr); err != nil {
		return fmt.Errorf("prepare send payment: %w", err)
	}

	pm, ok := prepResp.PaymentMethod.(spark.SendPaymentMethodBolt11Invoice)
	if !ok {
		return fmt.Errorf("unexpected payment method %T", prepResp.PaymentMethod)
	}
	if pm.InvoiceDetails.AmountMsat == nil {
		return fmt.Errorf("zero-amount invoices are not supported")
	}
	estimateFeeMsat := int64(pm.LightningFeeSats) * 1000
	invoiceAmountMsat := int64(*pm.InvoiceDetails.AmountMsat)

	// bolt11 invoices carry their own amount, which always wins.
	amountMsat := invoiceAmountMsat
	if requestedMsat != 0 && invoiceAmountMsat != requestedMsat {
		return fmt.Errorf("invoice amount %d msat does not match requested %d msat", invoiceAmountMsat, requestedMsat)
	}
	if _, err := resolveWithdrawAmount(amountMsat, capMsat, estimateFeeMsat); err != nil {
		return err
	}

	id, err := srv.insertOperatorTx(operatorKindWithdraw, amountMsat, 0, invoice.Field0.Invoice.Bolt11, "")
	if err != nil {
		return fmt.Errorf("insert operator withdraw: %w", err)
	}
	idempotencyKey := uuid5(satoshiNoteUUIDNamespace, fmt.Sprintf("operator-withdraw:%d", id))
	if err := srv.setOperatorTxIdempotencyKey(id, idempotencyKey); err != nil {
		slog.Error("store operator withdraw idempotency key", "id", id, "err", err)
	}

	sendResp, rawSendErr := srv.ln.SendPayment(spark.SendPaymentRequest{
		PrepareResponse: prepResp,
		IdempotencyKey:  &idempotencyKey,
	})
	if sendErr := sdkErr(rawSendErr); sendErr != nil {
		srv.failOperatorTx(id, sendErr.Error())
		return fmt.Errorf("send payment: %w", sendErr)
	}

	var actualFeeMsat int64
	if sendResp.Payment.Fees != nil {
		actualFeeMsat = sendResp.Payment.Fees.Int64() * 1000
	}
	if err := srv.setOperatorTxPaymentID(id, sendResp.Payment.Id); err != nil {
		slog.Error("store operator withdraw payment id", "id", id, "err", err)
	}
	if sendResp.Payment.Status != spark.PaymentStatusCompleted {
		slog.Warn("admin withdraw returned non-completed status",
			"id", id, "payment_id", sendResp.Payment.Id, "status", sendResp.Payment.Status)
	}

	hash, preimage := paymentHashPreimage(sendResp.Payment.Details)
	if err := srv.markOperatorTxConfirmed(id, actualFeeMsat, hash, preimage); err != nil {
		slog.Error("mark operator withdraw confirmed", "id", id, "err", err)
	}

	writeJSON(w, http.StatusOK, map[string]any{"status": "OK", "amount_msat": amountMsat, "fee_msat": actualFeeMsat})
	return nil
}

func (srv *Server) sendAdminWithdrawLnurl(w http.ResponseWriter, requestedMsat, capMsat int64, payRequest spark.LnurlPayRequestDetails) error {
	// Estimate the fee at the cap (full withdraw) or the requested amount, then
	// subtract it from the cap so a blank amount withdraws everything minus the fee.
	estimateBasis := requestedMsat
	if estimateBasis == 0 {
		estimateBasis = capMsat
	}
	estimate, err := srv.estimateLnurlFee(payRequest, estimateBasis)
	if err != nil {
		return err
	}

	amountMsat, err := resolveWithdrawAmount(requestedMsat, capMsat, estimate)
	if err != nil {
		return err
	}

	// Re-prepare at the final amount so the fee quote matches what we send.
	prepResp, err := srv.prepareLnurl(payRequest, amountMsat)
	if err != nil {
		return err
	}
	finalFeeMsat := int64(prepResp.FeeSats) * 1000
	if amountMsat+finalFeeMsat > capMsat {
		return fmt.Errorf("amount + routing fee %d msat exceeds withdrawable equity %d msat", amountMsat+finalFeeMsat, capMsat)
	}

	id, err := srv.insertOperatorTx(operatorKindWithdraw, amountMsat, 0, "", payRequest.Callback)
	if err != nil {
		return fmt.Errorf("insert operator withdraw: %w", err)
	}
	idempotencyKey := uuid5(satoshiNoteUUIDNamespace, fmt.Sprintf("operator-withdraw:%d", id))
	if err := srv.setOperatorTxIdempotencyKey(id, idempotencyKey); err != nil {
		slog.Error("store operator withdraw idempotency key", "id", id, "err", err)
	}

	lnurlPayResp, rawPayErr := srv.ln.LnurlPay(spark.LnurlPayRequest{
		PrepareResponse: prepResp,
		IdempotencyKey:  &idempotencyKey,
	})
	if err := sdkErr(rawPayErr); err != nil {
		srv.failOperatorTx(id, err.Error())
		return fmt.Errorf("lnurl pay: %w", err)
	}

	var actualFeeMsat int64
	if lnurlPayResp.Payment.Fees != nil {
		actualFeeMsat = lnurlPayResp.Payment.Fees.Int64() * 1000
	}
	if err := srv.setOperatorTxPaymentID(id, lnurlPayResp.Payment.Id); err != nil {
		slog.Error("store operator withdraw payment id", "id", id, "err", err)
	}
	if lnurlPayResp.Payment.Status != spark.PaymentStatusCompleted {
		slog.Warn("admin withdraw lnurl returned non-completed status",
			"id", id, "payment_id", lnurlPayResp.Payment.Id, "status", lnurlPayResp.Payment.Status)
	}

	hash, preimage := paymentHashPreimage(lnurlPayResp.Payment.Details)
	if err := srv.markOperatorTxConfirmed(id, actualFeeMsat, hash, preimage); err != nil {
		slog.Error("mark operator withdraw confirmed", "id", id, "err", err)
	}

	writeJSON(w, http.StatusOK, map[string]any{"status": "OK", "amount_msat": amountMsat, "fee_msat": actualFeeMsat})
	return nil
}

// estimateLnurlFee returns the SDK's routing-fee quote (msat) for paying amountMsat.
func (srv *Server) estimateLnurlFee(payRequest spark.LnurlPayRequestDetails, amountMsat int64) (int64, error) {
	prepResp, err := srv.prepareLnurl(payRequest, amountMsat)
	if err != nil {
		return 0, err
	}
	return int64(prepResp.FeeSats) * 1000, nil
}

func (srv *Server) prepareLnurl(payRequest spark.LnurlPayRequestDetails, amountMsat int64) (spark.PrepareLnurlPayResponse, error) {
	amountSats := uint64(amountMsat / 1000)
	amount := new(big.Int).SetUint64(amountSats)
	comment := "Operator equity withdrawal"

	var commentPtr *string
	if payRequest.CommentAllowed > 0 {
		if len(comment) > int(payRequest.CommentAllowed) {
			truncated := comment[:payRequest.CommentAllowed]
			commentPtr = &truncated
		} else {
			commentPtr = &comment
		}
	}

	prepResp, rawPrepErr := srv.ln.PrepareLnurlPay(spark.PrepareLnurlPayRequest{
		Amount:     amount,
		PayRequest: payRequest,
		Comment:    commentPtr,
	})
	if err := sdkErr(rawPrepErr); err != nil {
		return spark.PrepareLnurlPayResponse{}, fmt.Errorf("prepare lnurl pay: %w", err)
	}
	return prepResp, nil
}

// failOperatorTx marks a withdraw failed and logs if that itself fails.
func (srv *Server) failOperatorTx(id int64, msg string) {
	if err := srv.markOperatorTxFailed(id, msg); err != nil {
		slog.Error("mark operator tx failed", "id", id, "err", err)
	}
}

// paymentHashPreimage extracts the lightning hash/preimage from payment details.
func paymentHashPreimage(details *spark.PaymentDetails) (string, string) {
	if details == nil {
		return "", ""
	}
	if d, ok := (*details).(spark.PaymentDetailsLightning); ok {
		preimage := ""
		if d.HtlcDetails.Preimage != nil {
			preimage = *d.HtlcDetails.Preimage
		}
		return d.HtlcDetails.PaymentHash, preimage
	}
	return "", ""
}

// --- operator_txs persistence ---

func (srv *Server) insertOperatorTx(kind string, amountMsat, feeMsat int64, pr, destination string) (int64, error) {
	res, err := srv.db.Exec(
		`INSERT INTO operator_txs (kind, status, amount_msat, fee_msat, pr, destination, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		kind, TxPending, amountMsat, feeMsat, pr, destination, time.Now().Unix(),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (srv *Server) setOperatorTxIdempotencyKey(id int64, key string) error {
	_, err := srv.db.Exec(
		`UPDATE operator_txs SET idempotency_key = ?, updated_at = ? WHERE id = ?`,
		key, time.Now().Unix(), id,
	)
	return err
}

func (srv *Server) setOperatorTxPaymentID(id int64, paymentID string) error {
	_, err := srv.db.Exec(
		`UPDATE operator_txs SET payment_id = ?, updated_at = ? WHERE id = ?`,
		paymentID, time.Now().Unix(), id,
	)
	return err
}

// markOperatorTxConfirmed is idempotent: only a pending row flips, so a duplicate
// payment-succeeded event plus the startup catch-up cannot double-confirm.
func (srv *Server) markOperatorTxConfirmed(id, feeMsat int64, paymentHash, paymentPreimage string) error {
	res, err := srv.db.Exec(
		`UPDATE operator_txs SET status = ?, fee_msat = ?, payment_hash = ?, payment_preimage = ?, updated_at = ?
		 WHERE id = ? AND status = ?`,
		TxConfirmed, feeMsat, paymentHash, paymentPreimage, time.Now().Unix(), id, TxPending,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		slog.Info("operator tx already left pending, skipping confirm", "id", id)
	}
	return nil
}

func (srv *Server) markOperatorTxFailed(id int64, errMsg string) error {
	_, err := srv.db.Exec(
		`UPDATE operator_txs SET status = ?, error_msg = ?, updated_at = ? WHERE id = ?`,
		TxFailed, errMsg, time.Now().Unix(), id,
	)
	return err
}

// confirmOperatorDepositByPR marks a pending deposit confirmed when its invoice is
// paid. Idempotent via the status guard.
func (srv *Server) confirmOperatorDepositByPR(pr string, amountMsat int64, paymentHash, paymentPreimage string) error {
	res, err := srv.db.Exec(
		`UPDATE operator_txs SET status = ?, amount_msat = ?, payment_hash = ?, payment_preimage = ?, updated_at = ?
		 WHERE pr = ? AND kind = ? AND status = ?`,
		TxConfirmed, amountMsat, paymentHash, paymentPreimage, time.Now().Unix(), pr, operatorKindDeposit, TxPending,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

type operatorTxRow struct {
	ID             int64
	Kind           string
	AmountMsat     int64
	PR             string
	PaymentID      string
	IdempotencyKey string
	CreatedAt      int64
}

func (srv *Server) getPendingOperatorTxs(kind string) ([]operatorTxRow, error) {
	rows, err := srv.db.Query(
		`SELECT id, kind, amount_msat, pr, payment_id, idempotency_key, created_at
		 FROM operator_txs WHERE kind = ? AND status = ?`, kind, TxPending,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []operatorTxRow
	for rows.Next() {
		var t operatorTxRow
		if err := rows.Scan(&t.ID, &t.Kind, &t.AmountMsat, &t.PR, &t.PaymentID, &t.IdempotencyKey, &t.CreatedAt); err != nil {
			return nil, err
		}
		txs = append(txs, t)
	}
	return txs, rows.Err()
}
