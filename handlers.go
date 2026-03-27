package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	spark "github.com/breez/breez-sdk-spark-go/breez_sdk_spark"
)

func readPartial(name string) string {
	data, err := os.ReadFile("./static/" + name)
	if err != nil {
		return ""
	}
	return string(data)
}

func lnurlError(w http.ResponseWriter, reason string) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ERROR",
		"reason": reason,
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (srv *Server) handleCreateVouchers(w http.ResponseWriter, r *http.Request) {
	if !srv.cfg.createActive {
		lnurlError(w, "creating is currently disabled")
		return
	}

	var req struct {
		BatchName          string   `json:"batch_name"`
		PubKeys            []string `json:"pub_keys"`
		RefundCode         string   `json:"refund_code"`
		RefundAfterSeconds int64    `json:"refund_after_seconds"`
		SingleUse          bool     `json:"single_use"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		slog.Error("decode request body", "err", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.RefundCode = strings.ToLower(req.RefundCode)

	if req.RefundAfterSeconds <= 0 {
		http.Error(w, "refund_after_seconds must be greater than 0", http.StatusBadRequest)
		return
	}
	if req.RefundAfterSeconds > srv.cfg.maxVoucherExpireSeconds {
		req.RefundAfterSeconds = srv.cfg.maxVoucherExpireSeconds
	}

	if len(req.PubKeys) == 0 {
		http.Error(w, "pub_keys must not be empty", http.StatusBadRequest)
		return
	}
	if int64(len(req.PubKeys)) > srv.cfg.maxVouchersPerBatch {
		http.Error(w, "too many vouchers", http.StatusBadRequest)
		return
	}
	for _, pk := range req.PubKeys {
		b, err := hex.DecodeString(pk)
		if err != nil || len(b) < 16 || len(b) > 32 {
			http.Error(w, "invalid pub_key: must be hex, 16–32 bytes", http.StatusBadRequest)
			return
		}
	}

	batchIDBytes := make([]byte, srv.cfg.randomBytesLength)
	if _, err := rand.Read(batchIDBytes); err != nil {
		slog.Error("create batch id error", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	batchID := hex.EncodeToString(batchIDBytes[:srv.cfg.randomBytesLength])

	// Create all vouchers in a single DB transaction so a partial failure leaves no orphaned rows.
	dbTx, err := srv.db.Begin()
	if err != nil {
		slog.Error("begin transaction", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer dbTx.Rollback()

	var vs []Voucher
	for _, pubKey := range req.PubKeys {
		voucher := srv.newVoucher(pubKey, req.RefundCode, req.BatchName, batchID, req.RefundAfterSeconds, req.SingleUse)

		if _, err := dbTx.Exec(
			`INSERT INTO vouchers (pub_key, batch_name, batch_id, refund_code, refund_after_seconds, single_use, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			voucher.PubKey, voucher.BatchName, voucher.BatchID,
			voucher.RefundCode, voucher.RefundAfterSeconds, boolToInt(voucher.SingleUse),
			time.Now().Unix(),
		); err != nil {
			slog.Error("insert voucher", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		vs = append(vs, *voucher)
	}

	if err := dbTx.Commit(); err != nil {
		slog.Error("commit vouchers", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(vs); err != nil {
		slog.Error("encode response", "err", err)
	}
}

// GET /f/{pubKey} — LNURL-pay step 1
func (srv *Server) handleLNURLPayVoucher(w http.ResponseWriter, r *http.Request) {
	if !srv.cfg.fundActive {
		lnurlError(w, "funding is currently disabled")
		return
	}

	key := r.PathValue("pubKey")

	if key == "donate" {
		writeJSON(w, http.StatusOK, lnurlPayResponse(
			"Donate to "+srv.cfg.siteName,
			srv.cfg.baseURL+"/fund/donate/callback",
			srv.cfg.minFundAmountMsat, srv.cfg.maxFundAmountMsat,
			map[string]any{"commentAllowed": 200},
		))
		return
	}

	if _, err := srv.getVoucherByPubKey(srv.db, key); err == nil {
		writeJSON(w, http.StatusOK, lnurlPayResponse(
			"Fund a "+srv.cfg.siteName+" Voucher: "+key,
			srv.cfg.baseURL+"/fund/"+key+"/callback",
			srv.cfg.minFundAmountMsat, srv.cfg.maxFundAmountMsat,
			nil,
		))
		return
	}

	vs, err := srv.getVouchersByBatchID(srv.db, key)
	if err != nil {
		lnurlError(w, "voucher or batch not found")
		return
	}

	n := int64(len(vs))
	writeJSON(w, http.StatusOK, lnurlPayResponse(
		"Fund "+srv.cfg.siteName+" Vouchers ("+strconv.Itoa(len(vs))+") - Batch: "+key,
		srv.cfg.baseURL+"/fund/"+key+"/callback",
		srv.cfg.minFundAmountMsat*n, srv.cfg.maxFundAmountMsat*n,
		nil,
	))
}

// GET /fund/{pubKey}/callback?amount=MSATS — LNURL-pay step 2
func (srv *Server) handleLNURLPayCallbackVoucher(w http.ResponseWriter, r *http.Request) {
	if !srv.cfg.fundActive {
		lnurlError(w, "funding is currently disabled")
		return
	}

	key := r.PathValue("pubKey")

	if key == "donate" {
		amountMsat, err := srv.getCallbackAmount(r)
		if err != nil {
			lnurlError(w, "invalid amount")
			return
		}
		comment := r.URL.Query().Get("comment")
		if len(comment) > 200 {
			comment = comment[:200]
		}
		sat := amountMsat / 1000
		amountMsat = sat * 1000
		usat, err := Int64ToUint64(sat)
		if err != nil {
			lnurlError(w, "invalid amount")
			return
		}
		uexpiry, err := Int64ToUint32(srv.cfg.invoiceExpirySeconds)
		if err != nil {
			lnurlError(w, "internal error")
			return
		}
		resp, rawErr := srv.ln.ReceivePayment(spark.ReceivePaymentRequest{
			PaymentMethod: spark.ReceivePaymentMethodBolt11Invoice{
				AmountSats:  &usat,
				Description: "Donate to " + srv.cfg.siteName,
				ExpirySecs:  &uexpiry,
			},
		})
		if err := sdkErr(rawErr); err != nil {
			slog.Error("create donation invoice", "err", err)
			lnurlError(w, "failed to create invoice")
			return
		}
		var feeMsat int64
		if resp.Fee != nil {
			feeMsat = resp.Fee.Int64() * 1000
		}
		pr := resp.PaymentRequest
		donKey, err := srv.insertDonation(pr, amountMsat, feeMsat, comment)
		if err != nil {
			slog.Error("insert donation", "err", err)
			lnurlError(w, "internal error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "OK",
			"pr":     pr,
			"routes": []any{},
			"verify": srv.cfg.baseURL + "/verify/" + donKey,
		})
		return
	}

	tx := &FundTx{}

	if _, err := srv.getVoucherByPubKey(srv.db, key); err == nil {
		tx.PubKey = key
		var err error
		tx.Msat, err = srv.getCallbackAmount(r)
		if err != nil {
			lnurlError(w, "invalid amount")
			return
		}
		if err = srv.getCallbackBolt11(tx, "Fund a "+srv.cfg.siteName+" Voucher: "+key); err != nil {
			slog.Error("create invoice", "err", err)
			lnurlError(w, "failed to create invoice")
			return
		}
	} else {
		vs, err := srv.getVouchersByBatchID(srv.db, key)
		if err != nil {
			lnurlError(w, "voucher or batch not found")
			return
		}
		tx.BatchID = key
		msatsStr := r.URL.Query().Get("amount")
		if msatsStr == "" {
			lnurlError(w, "missing amount")
			return
		}
		msats, err := strconv.ParseInt(msatsStr, 10, 64)
		if err != nil {
			lnurlError(w, "invalid amount")
			return
		}
		batchMin := srv.cfg.minFundAmountMsat * int64(len(vs))
		batchMax := srv.cfg.maxFundAmountMsat * int64(len(vs))
		if msats < batchMin || msats > batchMax {
			lnurlError(w, "amount out of range")
			return
		}
		tx.Msat = msats
		if err = srv.getCallbackBolt11(tx, "Fund "+srv.cfg.siteName+" Vouchers ("+strconv.Itoa(len(vs))+") - Batch: "+key); err != nil {
			slog.Error("create invoice", "err", err)
			lnurlError(w, "failed to create invoice")
			return
		}
	}

	if err := srv.insertFundTX(tx); err != nil {
		slog.Error("insert fund tx", "err", err)
		lnurlError(w, "failed to write fund tx")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status": "OK",
		"pr":     tx.PR,
		"routes": []any{},
		"verify": srv.cfg.baseURL + "/verify/" + tx.Key,
	})
}

// POST /transfer — move funds from a non-single-use voucher to any destination (pubKey, batchID, or "donate")
func (srv *Server) handleTransfer(w http.ResponseWriter, r *http.Request) {
	if !srv.cfg.fundActive || !srv.cfg.redeemActive {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "transfers are currently disabled"})
		return
	}

	var req struct {
		Secret     string `json:"secret"`
		PubKey     string `json:"pub_key"`
		AmountMsat int64  `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Secret == "" || req.PubKey == "" || req.AmountMsat <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "secret, pub_key, and amount are required"})
		return
	}

	// Resolve source voucher
	srcPubKey, err := secretToPubKey(req.Secret)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid secret"})
		return
	}
	src, err := srv.getVoucherByPubKey(srv.db, srcPubKey)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "source voucher not found"})
		return
	}
	if src.SingleUse {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "single-use vouchers cannot transfer funds"})
		return
	}

	// Resolve destination and validate minimum amount
	isDonate := req.PubKey == "donate"
	var dstPubKey, dstBatchID string
	var dstCount int64 = 1

	if !isDonate {
		if _, err := srv.getVoucherByPubKey(srv.db, req.PubKey); err == nil {
			if req.PubKey == srcPubKey {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "source and destination cannot be the same voucher"})
				return
			}
			dstPubKey = req.PubKey
		} else {
			vs, err := srv.getVouchersByBatchID(srv.db, req.PubKey)
			if err != nil {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "destination not found"})
				return
			}
			dstBatchID = req.PubKey
			dstCount = int64(len(vs))
		}
	}

	if req.AmountMsat < srv.cfg.minFundAmountMsat*dstCount {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "amount below minimum"})
		return
	}

	// Calculate fee (rounded down to nearest sat)
	feeMsat := req.AmountMsat * srv.cfg.internalFeeBPS / 10000 / 1000 * 1000
	if feeMsat < srv.cfg.minInternalFeeMsat {
		feeMsat = srv.cfg.minInternalFeeMsat / 1000 * 1000
	}
	netMsat := req.AmountMsat - feeMsat

	if req.AmountMsat > src.BalanceMsat {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "insufficient balance"})
		return
	}

	// Execute atomically
	dbTx, err := srv.db.Begin()
	if err != nil {
		slog.Error("transfer begin tx", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	defer dbTx.Rollback()

	// Deduct from source
	if err := srv.updateVoucherBalance(dbTx, src.ID, -req.AmountMsat); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "insufficient balance"})
		return
	}

	// Credit destination
	var dustMsat int64
	if isDonate {
		// funds remain in node wallet; audit record inserted below
	} else if dstPubKey != "" {
		dst, err := srv.getVoucherByPubKey(dbTx, dstPubKey)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "destination voucher not found"})
			return
		}
		if err := srv.updateVoucherBalance(dbTx, dst.ID, netMsat); err != nil {
			slog.Error("transfer credit destination", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to credit destination"})
			return
		}
	} else {
		vs, err := srv.getVouchersByBatchID(dbTx, dstBatchID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "destination batch not found"})
			return
		}
		share := netMsat / int64(len(vs)) / 1000 * 1000
		dustMsat = netMsat - share*int64(len(vs))
		for _, v := range vs {
			if err := srv.updateVoucherBalance(dbTx, v.ID, share); err != nil {
				slog.Error("transfer credit batch voucher", "err", err)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to credit destination"})
				return
			}
		}
	}

	if err := srv.insertTransferTx(dbTx, srcPubKey, dstPubKey, dstBatchID, isDonate, req.AmountMsat, feeMsat, netMsat, dustMsat); err != nil {
		slog.Error("insert transfer tx", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to record transfer"})
		return
	}

	if err := dbTx.Commit(); err != nil {
		slog.Error("transfer commit", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"amount": req.AmountMsat,
		"fee_msat":    feeMsat,
		"net_msat":    netMsat,
	})
}

func (srv *Server) handleLNURLVerify(w http.ResponseWriter, r *http.Request) {
	if !srv.cfg.fundActive {
		lnurlError(w, "funding is currently disabled")
		return
	}

	key := r.PathValue("key")

	if tx, err := srv.getFundTxByKey(key); err == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":   "OK",
			"settled":  tx.Status == TxConfirmed,
			"preimage": tx.PaymentPreimage,
			"pr":       tx.PR,
		})
		return
	}

	don, err := srv.getDonationByKey(key)
	if err != nil {
		lnurlError(w, "not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "OK",
		"settled":  don.Status == TxConfirmed,
		"preimage": don.PaymentPreimage,
		"pr":       don.PR,
	})
}

func (srv *Server) handleLNURLWithdraw(w http.ResponseWriter, r *http.Request) {
	if !srv.cfg.redeemActive {
		lnurlError(w, "redeem is currently disabled")
		return
	}

	secret := r.PathValue("secret")

	pubKey, err := secretToPubKey(secret)
	if err != nil {
		lnurlError(w, "invalid secret")
		return
	}

	v, err := srv.getVoucherByPubKey(srv.db, pubKey)
	if err != nil {
		lnurlError(w, "voucher not found")
		return
	}

	if v.BalanceMsat < int64(srv.cfg.minRedeemAmountMsat) {
		lnurlError(w, "voucher balance too low")
		return
	}

	k1Bytes := make([]byte, srv.cfg.randomBytesLength)
	if _, err := rand.Read(k1Bytes); err != nil {
		lnurlError(w, "internal error")
		return
	}
	k1 := hex.EncodeToString(k1Bytes)

	err = srv.insertRedeemSession(k1, pubKey)
	if err != nil {
		slog.Error("insert redeem session", "err", err)
		lnurlError(w, "internal error")
		return
	}

	dbTxFee := srv.calculateRedeemFee(v.BalanceMsat)

	maxRedeemable := v.BalanceMsat - dbTxFee
	minRedeemable := int64(srv.cfg.minRedeemAmountMsat) / 1000 * 1000

	if minRedeemable > maxRedeemable {
		lnurlError(w, "voucher balance too low")
		return
	}

	if v.SingleUse {
		minRedeemable = maxRedeemable
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"tag":                "withdrawRequest",
		"callback":           srv.cfg.baseURL + "/redeem/" + secret + "/callback",
		"k1":                 k1,
		"minWithdrawable":    minRedeemable,
		"maxWithdrawable":    maxRedeemable,
		"defaultDescription": "Redeem Voucher from " + srv.cfg.siteName,
	})
}

func (srv *Server) handleLNURLWithdrawCallback(w http.ResponseWriter, r *http.Request) {
	if !srv.cfg.redeemActive {
		lnurlError(w, "redeem is currently disabled")
		return
	}

	secret := r.PathValue("secret")

	pubKey, err := secretToPubKey(secret)
	if err != nil {
		lnurlError(w, "invalid secret")
		return
	}

	k1 := r.URL.Query().Get("k1")
	if k1 == "" {
		lnurlError(w, "missing k1")
		return
	}

	pr := r.URL.Query().Get("pr")
	if pr == "" {
		lnurlError(w, "missing pr")
		return
	}

	if !srv.paymentSema.acquireForWithdrawal() {
		lnurlError(w, "server busy, please retry")
		return
	}
	defer srv.paymentSema.releaseAfter(srv.cfg.paymentCooldown)

	err = srv.markRedeemSessionUsed(k1, pubKey)
	if err != nil {
		lnurlError(w, "invalid or expired k1")
		return
	}

	prepResp, rawPrepErr := srv.ln.PrepareSendPayment(spark.PrepareSendPaymentRequest{
		PaymentRequest: pr,
	})
	if err := sdkErr(rawPrepErr); err != nil {
		slog.Error("prepare send payment", "err", err)
		lnurlError(w, "prepare payment failed")
		return
	}

	var estimateFeeMsat int64
	var amountMsat int64
	switch pm := prepResp.PaymentMethod.(type) {
	case spark.SendPaymentMethodBolt11Invoice:
		if pm.InvoiceDetails.AmountMsat == nil {
			lnurlError(w, "zero-amount invoices are not supported")
			return
		}
		estimateFeeMsat = int64(pm.LightningFeeSats) * 1000
		amountMsat = int64(*pm.InvoiceDetails.AmountMsat)
	default:
		slog.Error("unsupported payment method", "type", fmt.Sprintf("%T", prepResp.PaymentMethod))
		lnurlError(w, "unsupported payment method")
		return
	}

	v, err := srv.getVoucherByPubKey(srv.db, pubKey)
	if err != nil {
		lnurlError(w, "voucher not found")
		return
	}

	dbTxFee := srv.calculateRedeemFee(v.BalanceMsat)

	if estimateFeeMsat > dbTxFee {
		lnurlError(w, "routing fee too high")
		return
	}

	if amountMsat+dbTxFee > v.BalanceMsat {
		lnurlError(w, "redeem amount exceeds voucher balance after fees")
		return
	}

	redeemID, err := srv.insertRedeemAndBalance(v, secret, pr, amountMsat, dbTxFee)
	if err != nil {
		slog.Error("insert redeem and balance", "err", err)
		lnurlError(w, "internal db error")
		return
	}

	sendResp, rawSendErr := srv.ln.SendPayment(spark.SendPaymentRequest{
		PrepareResponse: prepResp,
	})
	if sendErr := sdkErr(rawSendErr); sendErr != nil {
		slog.Error("send payment", "err", sendErr)
		if err := srv.updateRedeemTx(redeemID, TxFailed, 0, 0, sendErr.Error()); err != nil {
			slog.Error("update redeem tx failed", "err", err)
		}
		// Restore the balance that was deducted before the failed payment attempt.
		if err := srv.addVoucherBalance(v.ID, amountMsat+dbTxFee); err != nil {
			slog.Error("restore voucher balance after failed payment", "err", err)
		}
		lnurlError(w, "payment failed")
		return
	}

	var actualFeeMsat int64
	if sendResp.Payment.Fees != nil {
		actualFeeMsat = sendResp.Payment.Fees.Int64() * 1000
	}

	if err := srv.updateRedeemTx(redeemID, TxConfirmed, dbTxFee-actualFeeMsat, actualFeeMsat, ""); err != nil {
		slog.Error("update redeem tx confirmed", "err", err)
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "OK"})
}

func (srv *Server) insertRedeemAndBalance(v *Voucher, secret, pr string, msat, fee int64) (int64, error) {
	dbTx, err := srv.db.Begin()
	if err != nil {
		return 0, err
	}
	defer dbTx.Rollback()

	if err := srv.updateVoucherBalance(dbTx, v.ID, -msat-fee); err != nil {
		return 0, err
	}

	redeemID, err := srv.insertRedeemTx(dbTx, v.ID, secret, pr, msat, fee)
	if err != nil {
		return 0, err
	}

	return redeemID, dbTx.Commit()
}

func (srv *Server) updateFundTxConfirmed(tx *FundTx) error {
	dbTx, err := srv.db.Begin()
	if err != nil {
		return err
	}
	defer dbTx.Rollback()

	if err := updateFundTXStatus(dbTx, tx.Key, TxConfirmed, tx.PaymentHash, tx.PaymentPreimage); err != nil {
		slog.Error("update fund tx status", "err", err)
		return err
	}

	dust, err := srv.updateFundBalance(dbTx, tx)
	if err != nil {
		slog.Error("update voucher balance", "err", err)
		return err
	}

	if dust > 0 {
		if err := updateFundTxDust(dbTx, tx.Key, dust); err != nil {
			slog.Error("update fund tx dust", "err", err)
			return err
		}
	}

	return dbTx.Commit()
}

// updateFundBalance credits voucher balances for a confirmed fund tx.
// For batch payments, returns the msat remainder lost to per-sat rounding
// (tracked in fund_txs.dust_msat). Always returns 0 for single-voucher payments.
func (srv *Server) updateFundBalance(dbTx *sql.Tx, tx *FundTx) (int64, error) {
	if tx.PubKey != "" {
		v, err := srv.getVoucherByPubKey(dbTx, tx.PubKey)
		if err != nil {
			return 0, fmt.Errorf("get voucher by pubkey: %w", err)
		}
		return 0, srv.updateVoucherBalance(dbTx, v.ID, int64(tx.Msat)-tx.FeeMsat)
	}

	vs, err := srv.getVouchersByBatchID(dbTx, tx.BatchID)
	if err != nil {
		return 0, fmt.Errorf("get vouchers by batch id: %w", err)
	}

	total := int64(tx.Msat) - tx.FeeMsat
	share := total / int64(len(vs))
	share = (share / 1000) * 1000
	dust := total - share*int64(len(vs))

	for _, v := range vs {
		if err := srv.updateVoucherBalance(dbTx, v.ID, share); err != nil {
			return 0, fmt.Errorf("update voucher balance: %w", err)
		}
	}

	return dust, nil
}

type leaderboardRank struct {
	Rank  int `json:"rank"`
	Total int `json:"total"`
}

func top3(dist map[string]int) []int {
	vals := make([]int, 0, len(dist))
	for _, v := range dist {
		vals = append(vals, v)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(vals)))
	if len(vals) > 3 {
		vals = vals[:3]
	}
	return vals
}

func rankIn(userCount int, dist map[string]int) leaderboardRank {
	rank := 1
	for _, cnt := range dist {
		if cnt > userCount {
			rank++
		}
	}
	return leaderboardRank{Rank: rank, Total: len(dist)}
}

func (srv *Server) handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FundedMonth     int `json:"funded_month"`
		FundedAllTime   int `json:"funded_all_time"`
		RedeemedMonth   int `json:"redeemed_month"`
		RedeemedAllTime int `json:"redeemed_all_time"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.FundedMonth < 0 || req.FundedAllTime < 0 || req.RedeemedMonth < 0 || req.RedeemedAllTime < 0 {
		http.Error(w, "counts must be non-negative", http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Unix()

	fundedMonth, err := srv.leaderboardFundedMonth(monthStart)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	fundedAll, err := srv.leaderboardFundedAllTime()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	redeemedMonth, err := srv.leaderboardRedeemedMonth(monthStart)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	redeemedAll, err := srv.leaderboardRedeemedAllTime()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"funded_month":      rankIn(req.FundedMonth, fundedMonth),
		"funded_all_time":   rankIn(req.FundedAllTime, fundedAll),
		"redeemed_month":    rankIn(req.RedeemedMonth, redeemedMonth),
		"redeemed_all_time": rankIn(req.RedeemedAllTime, redeemedAll),
		"top_scores": map[string][]int{
			"funded_month":      top3(fundedMonth),
			"funded_all_time":   top3(fundedAll),
			"redeemed_month":    top3(redeemedMonth),
			"redeemed_all_time": top3(redeemedAll),
		},
	})
}

func (srv *Server) handleVoucherStatusBatch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PubKeys []string `json:"pubkeys"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if len(req.PubKeys) > 500 {
		http.Error(w, "too many pubkeys", http.StatusBadRequest)
		return
	}

	statuses, err := srv.getVoucherStatusBatch(req.PubKeys)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	result := make(map[string]any, len(req.PubKeys))
	for _, pubKey := range req.PubKeys {
		s, ok := statuses[pubKey]
		if !ok {
			continue
		}
		result[pubKey] = srv.voucherStatusBody(s)
	}

	writeJSON(w, http.StatusOK, result)
}

func (srv *Server) handleVoucherStatus(w http.ResponseWriter, r *http.Request) {
	pubKey := r.PathValue("pubKey")
	s, err := srv.getVoucherStatusByPubKey(pubKey)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, srv.voucherStatusBody(s))
}

func (srv *Server) getCallbackAmount(r *http.Request) (int64, error) {
	msatsStr := r.URL.Query().Get("amount")

	if msatsStr == "" {
		return 0, fmt.Errorf("missing amount")
	}
	msats, err := strconv.ParseInt(msatsStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount")
	}
	if msats < srv.cfg.minFundAmountMsat || msats > srv.cfg.maxFundAmountMsat {
		return 0, fmt.Errorf("amount out of range")
	}
	return msats, nil
}

func (srv *Server) handleRedeemPage(w http.ResponseWriter, r *http.Request) {
	srv.renderPage(w, "redeem.html", map[string]string{
		"{{HEADER_EXTRA}}": `<a href="/" style="font-size:0.75rem;color:var(--text-muted);text-decoration:none;white-space:nowrap;">Gift Bitcoin →</a>`,
	})
}

func (srv *Server) handleIndexPage(w http.ResponseWriter, r *http.Request) {
	srv.renderPage(w, "index.html", map[string]string{
		"{{FOOTER}}":           readPartial("footer.html"),
		"{{HEADER_EXTRA}}":     `<div style="display:flex;align-items:center;gap:8px;"><div id="wallet-balance-pill" class="balance-pill hidden">⚡ <span id="wallet-balance-sats">0</span> sats</div><button id="nav-menu" class="nav-btn hidden" aria-label="Menu" style="font-size:1.3rem;padding:4px 8px;">☰</button></div>`,
		"{{BATCH_ENABLED}}":    strconv.FormatBool(srv.cfg.batchEnabled),
		"{{DEFAULT_DIAL_CODE}}": srv.cfg.defaultDialCode,
	})
}


func (srv *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"random_bytes_length":  srv.cfg.randomBytesLength,
		"min_fund_amount_msat": srv.cfg.minFundAmountMsat,
	})
}

func (srv *Server) handleAdminStats(w http.ResponseWriter, r *http.Request) {
	if !srv.requireAdmin(w, r) {
		return
	}
	stats, err := srv.getAuditStats()
	if err != nil {
		slog.Error("get audit stats", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (srv *Server) handleAdminRecent(w http.ResponseWriter, r *http.Request) {
	if !srv.requireAdmin(w, r) {
		return
	}
	redeems, err := srv.getRecentRedeemTxs(10)
	if err != nil {
		slog.Error("get recent redeem txs", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	refunds, err := srv.getRecentRefundTxs(10)
	if err != nil {
		slog.Error("get recent refund txs", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if redeems == nil {
		redeems = []RedeemTx{}
	}
	if refunds == nil {
		refunds = []RefundTx{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"redeems": redeems,
		"refunds": refunds,
	})
}

func (srv *Server) handleAdminPage(w http.ResponseWriter, r *http.Request) {
	srv.renderPage(w, "admin.html", map[string]string{
		"{{FOOTER}}":       readPartial("footer.html"),
		"{{HEADER_EXTRA}}": `<div class="refresh-row"><span id="last-refresh-label"></span><button class="btn-refresh" id="btn-refresh">Refresh</button><button class="btn-refresh" id="btn-logout" style="color:var(--text-muted);">Logout</button></div>`,
	})
}

func (srv *Server) handleManifest(w http.ResponseWriter, r *http.Request) {
	b, err := os.ReadFile("./static/manifest.json")
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	out := strings.ReplaceAll(string(b), "{{SITE_NAME_FULL}}", srv.cfg.siteName)
	w.Header().Set("Content-Type", "application/manifest+json")
	_, _ = w.Write([]byte(out))
}

func (srv *Server) getAuditStats() (*AuditStats, error) {
	s := &AuditStats{}

	if err := srv.db.QueryRow(`SELECT COUNT(*) FROM vouchers`).Scan(&s.TotalVouchers); err != nil {
		return nil, err
	}
	if err := srv.db.QueryRow(`SELECT COUNT(*) FROM vouchers WHERE active=1 AND balance_msat=0`).Scan(&s.ActiveUnfunded); err != nil {
		return nil, err
	}
	if err := srv.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(balance_msat),0) FROM vouchers WHERE active=1 AND balance_msat>0`).Scan(&s.ActiveFunded, &s.ClaimableMsat); err != nil {
		return nil, err
	}
	if err := srv.db.QueryRow(`SELECT COUNT(*) FROM vouchers WHERE active=0 AND refunded=0`).Scan(&s.TotalRedeemed); err != nil {
		return nil, err
	}
	if err := srv.db.QueryRow(`SELECT COUNT(*) FROM vouchers WHERE active=0 AND refunded=1`).Scan(&s.TotalRefunded); err != nil {
		return nil, err
	}
	if err := srv.db.QueryRow(`SELECT COUNT(*) FROM refund_txs WHERE refunded=0 AND error_msg != ''`).Scan(&s.FailedRefunds); err != nil {
		return nil, err
	}
	if err := srv.db.QueryRow(`SELECT COALESCE(SUM(msat),0) FROM redeem_txs WHERE status='confirmed'`).Scan(&s.RedeemedMsat); err != nil {
		return nil, err
	}
	if err := srv.db.QueryRow(`SELECT COALESCE(SUM(amount_msat),0) FROM refund_txs WHERE refunded=1`).Scan(&s.RefundedMsat); err != nil {
		return nil, err
	}
	if err := srv.db.QueryRow(`SELECT COALESCE(SUM(amount_msat),0) FROM refund_txs WHERE refunded=0 AND refund_code != ''`).Scan(&s.PendingRefundMsat); err != nil {
		return nil, err
	}
	if err := srv.db.QueryRow(`SELECT COALESCE(SUM(msat),0) FROM fund_txs WHERE status='confirmed'`).Scan(&s.TotalDepositedMsat); err != nil {
		return nil, err
	}
	if err := srv.db.QueryRow(`SELECT COALESCE(SUM(dust_msat),0) FROM fund_txs WHERE status='confirmed'`).Scan(&s.TotalDustMsat); err != nil {
		return nil, err
	}
	if err := srv.db.QueryRow(`SELECT COALESCE(SUM(amount_msat),0) FROM refund_txs WHERE refund_code=''`).Scan(&s.ExpiredNoRefundMsat); err != nil {
		return nil, err
	}

	infoResp, err := srv.ln.GetInfo(spark.GetInfoRequest{})
	if sdkErr(err) == nil {
		s.BreezBalanceMsat = int64(infoResp.BalanceSats) * 1000
	} else {
		s.BreezBalanceMsat = -1
	}

	if s.BreezBalanceMsat >= 0 {
		s.SurplusMsat = s.BreezBalanceMsat - s.ClaimableMsat - s.PendingRefundMsat
	} else {
		s.SurplusMsat = -1
	}

	var redeemFees, refundFees int64
	if err := srv.db.QueryRow(`SELECT COALESCE(SUM(db_tx_fee),0) FROM redeem_txs WHERE status='confirmed'`).Scan(&redeemFees); err != nil {
		return nil, err
	}
	if err := srv.db.QueryRow(`SELECT COALESCE(SUM(db_tx_fee),0) FROM refund_txs WHERE refunded=1`).Scan(&refundFees); err != nil {
		return nil, err
	}
	s.TotalDbFeeMsat = redeemFees + refundFees

	s.TotalDonations, s.ConfirmedDonations, s.DonatedMsat, err = srv.getDonationStats()
	if err != nil {
		return nil, err
	}

	return s, nil
}
