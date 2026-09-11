package main

import (
	"fmt"
	"log/slog"
	"time"

	spark "github.com/breez/breez-sdk-spark-go/breez_sdk_spark"
)

const fundTxLookbackSecs = 600

// resolvePendingRedeemTxs inspects redeem_txs left in TxPending state from a previous
// run. A pending row means the server crashed after deducting the voucher balance but
// before (or during) SendPayment — the payment outcome is unknown.
//
// Rows with a stored SDK payment id are resolved via GetPayment: a completed payment
// is marked confirmed; a failed payment is marked failed and the deducted balance is
// restored. Rows without a payment id (crash before SendPayment returned) are NOT
// resolved automatically — the balance stays deducted pending manual review. A manual
// retry of such a redeem is safe: SendPayment is called with a per-redeem idempotency
// key, so the SDK returns the original payment instead of paying twice.
func (srv *Server) resolvePendingRedeemTxs() {
	rows, err := srv.db.Query(
		`SELECT id, voucher_id, msat, ln_fee, payment_id, created_at FROM redeem_txs WHERE status = ?`, TxPending,
	)
	if err != nil {
		slog.Error("startup: check pending redeem txs", "err", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id, voucherID, msat, lnFee, createdAt int64
		var paymentID string
		if err := rows.Scan(&id, &voucherID, &msat, &lnFee, &paymentID, &createdAt); err != nil {
			slog.Error("startup: scan pending redeem tx", "err", err)
			return
		}

		if paymentID == "" {
			slog.Warn("startup: found pending redeem tx from previous run — payment outcome unknown, manual review required",
				"id", id,
				"voucher_id", voucherID,
				"amount_msat", msat,
				"created_at", createdAt,
			)
			continue
		}

		resp, rawErr := srv.ln.GetPayment(spark.GetPaymentRequest{PaymentId: paymentID})
		if err := sdkErr(rawErr); err != nil {
			slog.Warn("startup: could not resolve pending redeem tx via GetPayment — manual review required",
				"id", id, "payment_id", paymentID, "err", err)
			continue
		}

		switch resp.Payment.Status {
		case spark.PaymentStatusCompleted:
			var actualFeeMsat int64
			if resp.Payment.Fees != nil {
				actualFeeMsat = resp.Payment.Fees.Int64() * 1000
			}
			if err := srv.updateRedeemTx(id, TxConfirmed, lnFee-actualFeeMsat, actualFeeMsat, ""); err != nil {
				slog.Error("startup: mark resolved redeem tx confirmed", "id", id, "err", err)
				continue
			}
			slog.Info("startup: resolved pending redeem tx as confirmed",
				"id", id, "payment_id", paymentID, "amount_msat", msat)
		case spark.PaymentStatusFailed:
			if err := srv.updateRedeemTx(id, TxFailed, 0, 0, "payment failed (resolved at startup)"); err != nil {
				slog.Error("startup: mark resolved redeem tx failed", "id", id, "err", err)
				continue
			}
			// Restore the balance deducted before the payment attempt.
			if err := srv.addVoucherBalance(voucherID, msat+lnFee); err != nil {
				slog.Error("startup: restore voucher balance for failed redeem", "id", id, "voucher_id", voucherID, "err", err)
				continue
			}
			slog.Info("startup: resolved pending redeem tx as failed, balance restored",
				"id", id, "payment_id", paymentID, "voucher_id", voucherID, "amount_msat", msat)
		default:
			slog.Warn("startup: pending redeem tx still in flight at the SDK — manual review required",
				"id", id, "payment_id", paymentID, "status", resp.Payment.Status)
		}
	}
}

func (srv *Server) checkPendingFundTXs() error {
	txs, err := srv.getPendingFundTxs()
	if err != nil {
		return err
	}

	if len(txs) == 0 {
		return nil
	}

	txMap := make(map[string]*FundTx)
	since := uint64(time.Now().Unix())
	for _, tx := range txs {
		txMap[tx.PR] = &tx

		if since > uint64(tx.CreatedAt) {
			since = uint64(tx.CreatedAt)
		}
	}

	ps, err := srv.getPaymentsCompleted(since - fundTxLookbackSecs) // Get payments prior to earliest pending
	if err != nil {
		return err
	}

	for _, p := range ps {
		if p.Details == nil {
			continue
		}

		if details, ok := (*p.Details).(spark.PaymentDetailsLightning); ok {
			if tx, yes := txMap[details.Invoice]; yes {
				tx.Msat = p.Amount.Int64() * 1000
				tx.FeeMsat = p.Fees.Int64() * 1000
				tx.PaymentHash = details.HtlcDetails.PaymentHash
				tx.PaymentPreimage = *details.HtlcDetails.Preimage

				if err := srv.updateFundTxConfirmed(tx); err != nil {
					return fmt.Errorf("update fund tx confirmed (pr=%s): %w", tx.PR, err)
				}
			}
		}
	}

	return nil
}

// checkPendingOperatorDeposits confirms operator deposit invoices that were paid
// while the server was down. Idempotent via the pending-status guard.
func (srv *Server) checkPendingOperatorDeposits() {
	txs, err := srv.getPendingOperatorTxs(operatorKindDeposit)
	if err != nil {
		slog.Error("startup: check pending operator deposits", "err", err)
		return
	}
	if len(txs) == 0 {
		return
	}

	byPR := make(map[string]operatorTxRow, len(txs))
	since := uint64(time.Now().Unix())
	for _, tx := range txs {
		byPR[tx.PR] = tx
		if since > uint64(tx.CreatedAt) {
			since = uint64(tx.CreatedAt)
		}
	}

	ps, err := srv.getPaymentsCompleted(since - fundTxLookbackSecs)
	if err != nil {
		slog.Error("startup: list payments for operator deposits", "err", err)
		return
	}

	for _, p := range ps {
		if p.Details == nil {
			continue
		}
		details, ok := (*p.Details).(spark.PaymentDetailsLightning)
		if !ok {
			continue
		}
		tx, yes := byPR[details.Invoice]
		if !yes {
			continue
		}
		var amountMsat int64
		if p.Amount != nil {
			amountMsat = p.Amount.Int64() * 1000
		}
		preimage := ""
		if details.HtlcDetails.Preimage != nil {
			preimage = *details.HtlcDetails.Preimage
		}
		if err := srv.confirmOperatorDepositByPR(details.Invoice, amountMsat, details.HtlcDetails.PaymentHash, preimage); err == nil {
			slog.Info("startup: confirmed operator deposit", "id", tx.ID, "amount_msat", amountMsat)
		}
	}
}

// resolvePendingOperatorWithdraws settles withdraw rows left pending by a crash.
// A row with a payment id is resolved via GetPayment (completed -> confirmed, failed
// -> failed). A row without one never reached the SDK send, so it is marked failed
// (no sats left; equity is not stuck).
func (srv *Server) resolvePendingOperatorWithdraws() {
	txs, err := srv.getPendingOperatorTxs(operatorKindWithdraw)
	if err != nil {
		slog.Error("startup: check pending operator withdraws", "err", err)
		return
	}
	for _, tx := range txs {
		if tx.PaymentID == "" {
			if err := srv.markOperatorTxFailed(tx.ID, "interrupted before payment was sent"); err != nil {
				slog.Error("startup: mark operator withdraw failed", "id", tx.ID, "err", err)
			}
			continue
		}

		resp, rawErr := srv.ln.GetPayment(spark.GetPaymentRequest{PaymentId: tx.PaymentID})
		if err := sdkErr(rawErr); err != nil {
			slog.Warn("startup: could not resolve operator withdraw via GetPayment — manual review required",
				"id", tx.ID, "payment_id", tx.PaymentID, "err", err)
			continue
		}

		switch resp.Payment.Status {
		case spark.PaymentStatusCompleted:
			var actualFeeMsat int64
			if resp.Payment.Fees != nil {
				actualFeeMsat = resp.Payment.Fees.Int64() * 1000
			}
			hash, preimage := paymentHashPreimage(resp.Payment.Details)
			if err := srv.markOperatorTxConfirmed(tx.ID, actualFeeMsat, hash, preimage); err != nil {
				slog.Error("startup: mark operator withdraw confirmed", "id", tx.ID, "err", err)
				continue
			}
			slog.Info("startup: resolved operator withdraw as confirmed", "id", tx.ID, "payment_id", tx.PaymentID)
		case spark.PaymentStatusFailed:
			if err := srv.markOperatorTxFailed(tx.ID, "payment failed (resolved at startup)"); err != nil {
				slog.Error("startup: mark operator withdraw failed", "id", tx.ID, "err", err)
				continue
			}
			slog.Info("startup: resolved operator withdraw as failed", "id", tx.ID, "payment_id", tx.PaymentID)
		default:
			slog.Warn("startup: operator withdraw still in flight at the SDK — manual review required",
				"id", tx.ID, "payment_id", tx.PaymentID, "status", resp.Payment.Status)
		}
	}
}
