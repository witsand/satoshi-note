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
	"strconv"
	"strings"
	"time"

	spark "github.com/breez/breez-sdk-spark-go/breez_sdk_spark"
)

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
		BatchName          string `json:"batch_name"`
		Amount             int64  `json:"amount"`
		RefundCode         string `json:"refund_code"`
		RefundAfterSeconds int64  `json:"refund_after_seconds"`
		SingleUse          bool   `json:"single_use"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		slog.Error("decode request body", "err", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.RefundAfterSeconds == 0 {
		http.Error(w, "refund_after_seconds must be greater than 0", http.StatusBadRequest)
		return
	}
	if req.RefundAfterSeconds > srv.cfg.maxVoucherExpireSeconds {
		req.RefundAfterSeconds = srv.cfg.maxVoucherExpireSeconds
	}

	if req.Amount > srv.cfg.maxVouchersPerBatch {
		http.Error(w, "too many vouchers requested", http.StatusBadRequest)
		return
	}
	if req.Amount == 0 {
		req.Amount = 1
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
	for i := int64(0); i < req.Amount; i++ {
		voucher, err := srv.newVoucher(req.RefundCode, req.BatchName, batchID, req.RefundAfterSeconds, req.SingleUse)
		if err != nil {
			slog.Error("create voucher", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		if _, err := dbTx.Exec(
			`INSERT INTO vouchers (secret, pub_key, batch_name, batch_id, refund_code, refund_after_seconds, single_use, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			voucher.Secret, voucher.PubKey, voucher.BatchName, voucher.BatchID,
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

	pubKey := r.PathValue("pubKey")

	_, err := srv.getVoucherByPubKey(srv.db, pubKey)
	if err != nil {
		lnurlError(w, "voucher not found")
		return
	}

	metadata := [][]string{{"text/plain", "Fund a " + srv.cfg.siteName + " Voucher: " + pubKey}}
	metaJSON, _ := json.Marshal(metadata)

	writeJSON(w, http.StatusOK, map[string]any{
		"tag":         "payRequest",
		"callback":    srv.cfg.baseURL + "/fund/" + pubKey + "/callback",
		"minSendable": srv.cfg.minFundAmountMsat,
		"maxSendable": srv.cfg.maxFundAmountMsat,
		"metadata":    string(metaJSON),
	})
}

// GET /fb/{batchID} — LNURL-pay step 1
func (srv *Server) handleLNURLPayBatch(w http.ResponseWriter, r *http.Request) {
	if !srv.cfg.fundActive {
		lnurlError(w, "funding is currently disabled")
		return
	}

	batchID := r.PathValue("batchID")

	vs, err := srv.getVouchersByBatchID(srv.db, batchID)
	if err != nil {
		lnurlError(w, "batch not found")
		return
	}

	metadata := [][]string{{"text/plain", "Fund " + srv.cfg.siteName + " Vouchers (" + strconv.Itoa(len(vs)) + ") - Batch: " + batchID}}
	metaJSON, _ := json.Marshal(metadata)

	writeJSON(w, http.StatusOK, map[string]any{
		"tag":         "payRequest",
		"callback":    srv.cfg.baseURL + "/fund/batch/" + batchID + "/callback",
		"minSendable": srv.cfg.minFundAmountMsat * int64(len(vs)),
		"maxSendable": srv.cfg.maxFundAmountMsat * int64(len(vs)),
		"metadata":    string(metaJSON),
	})
}

// GET /fund/{pubKey}/callback?amount=MSATS — LNURL-pay step 2
func (srv *Server) handleLNURLPayCallbackVoucher(w http.ResponseWriter, r *http.Request) {
	if !srv.cfg.fundActive {
		lnurlError(w, "funding is currently disabled")
		return
	}

	tx := &FundTx{}

	var err error
	tx.Msat, err = srv.getCallbackAmount(r)
	if err != nil {
		lnurlError(w, "invalid amount")
		return
	}

	tx.PubKey = r.PathValue("pubKey")
	_, err = srv.getVoucherByPubKey(srv.db, tx.PubKey)
	if err != nil {
		lnurlError(w, "voucher not found")
		return
	}

	if err = srv.getCallbackBolt11(tx, "Fund a "+srv.cfg.siteName+" Voucher: "+tx.PubKey); err != nil {
		slog.Error("create invoice", "err", err)
		lnurlError(w, "failed to create invoice")
		return
	}

	err = srv.insertFundTX(tx)
	if err != nil {
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

// GET /fund/batch/{batchID}/callback?amount=MSATS — LNURL-pay step 2
func (srv *Server) handleLNURLPayCallbackBatch(w http.ResponseWriter, r *http.Request) {
	if !srv.cfg.fundActive {
		lnurlError(w, "funding is currently disabled")
		return
	}

	tx := &FundTx{}
	tx.BatchID = r.PathValue("batchID")

	vs, err := srv.getVouchersByBatchID(srv.db, tx.BatchID)
	if err != nil {
		lnurlError(w, "batch not found")
		return
	}

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

	err = srv.getCallbackBolt11(tx, "Fund "+srv.cfg.siteName+" Vouchers ("+strconv.Itoa(len(vs))+") - Batch: "+tx.BatchID)
	if err != nil {
		slog.Error("create invoice", "err", err)
		lnurlError(w, "failed to create invoice")
		return
	}

	err = srv.insertFundTX(tx)
	if err != nil {
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

func (srv *Server) handleLNURLVerify(w http.ResponseWriter, r *http.Request) {
	if !srv.cfg.fundActive {
		lnurlError(w, "funding is currently disabled")
		return
	}

	key := r.PathValue("key")

	tx, err := srv.getFundTxByKey(key)
	if err != nil {
		lnurlError(w, "not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "OK",
		"settled":  tx.Status == TxConfirmed,
		"preimage": tx.PaymentPreimage,
		"pr":       tx.PR,
	})
}

func (srv *Server) handleLNURLWithdraw(w http.ResponseWriter, r *http.Request) {
	if !srv.cfg.redeemActive {
		lnurlError(w, "redeem is currently disabled")
		return
	}

	secret := r.PathValue("secret")

	v, err := srv.getVoucherBySecret(srv.db, secret)
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

	err = srv.insertRedeemSession(k1, secret)
	if err != nil {
		slog.Error("insert redeem session", "err", err)
		lnurlError(w, "internal error")
		return
	}

	dbTxFee := v.BalanceMsat*srv.cfg.redeemFeeBPS/10000/1000*1000 + 1000
	if dbTxFee < srv.cfg.minRedeemFeeMsat {
		dbTxFee = srv.cfg.minRedeemFeeMsat / 1000 * 1000
	}

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

	err := srv.markRedeemSessionUsed(k1, secret)
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

	v, err := srv.getVoucherBySecret(srv.db, secret)
	if err != nil {
		lnurlError(w, "voucher not found")
		return
	}

	dbTxFee := v.BalanceMsat*srv.cfg.redeemFeeBPS/10000/1000*1000 + 1000
	if dbTxFee < srv.cfg.minRedeemFeeMsat {
		dbTxFee = srv.cfg.minRedeemFeeMsat / 1000 * 1000
	}

	if estimateFeeMsat > dbTxFee {
		lnurlError(w, "routing fee too high")
		return
	}

	if amountMsat+dbTxFee > v.BalanceMsat {
		lnurlError(w, "redeem amount exceeds voucher balance after fees")
		return
	}

	redeemID, err := srv.insertRedeemAndBalance(v, pr, amountMsat, dbTxFee)
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

func (srv *Server) insertRedeemAndBalance(v *Voucher, pr string, msat, fee int64) (int64, error) {
	dbTx, err := srv.db.Begin()
	if err != nil {
		return 0, err
	}
	defer dbTx.Rollback()

	if err := srv.updateVoucherBalance(dbTx, v.ID, -msat-fee); err != nil {
		return 0, err
	}

	redeemID, err := srv.insertRedeemTx(dbTx, v.ID, pr, msat, fee)
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

	if err := srv.updateFundBalance(dbTx, tx); err != nil {
		slog.Error("update voucher balance", "err", err)
		return err
	}

	return dbTx.Commit()
}

func (srv *Server) updateFundBalance(dbTx *sql.Tx, tx *FundTx) error {
	if tx.PubKey != "" {
		v, err := srv.getVoucherByPubKey(dbTx, tx.PubKey)
		if err != nil {
			return fmt.Errorf("get voucher by pubkey: %w", err)
		}
		return srv.updateVoucherBalance(dbTx, v.ID, int64(tx.Msat)-tx.FeeMsat)
	}

	vs, err := srv.getVouchersByBatchID(dbTx, tx.BatchID)
	if err != nil {
		return fmt.Errorf("get vouchers by batch id: %w", err)
	}

	total := int64(tx.Msat) - tx.FeeMsat
	share := total / int64(len(vs))
	share = (share / 1000) * 1000

	for _, v := range vs {
		if err := srv.updateVoucherBalance(dbTx, v.ID, share); err != nil {
			return fmt.Errorf("update voucher balance: %w", err)
		}
	}

	return nil
}

func (srv *Server) handleVoucherStatus(w http.ResponseWriter, r *http.Request) {
	secret := r.PathValue("secret")
	s, err := srv.getVoucherStatusBySecret(secret)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	dbTxFee := s.BalanceMsat*srv.cfg.redeemFeeBPS/10000/1000*1000 + 1000
	if dbTxFee < srv.cfg.minRedeemFeeMsat {
		dbTxFee = srv.cfg.minRedeemFeeMsat / 1000 * 1000
	}

	var maxRedeemable int64
	if s.BalanceMsat > dbTxFee {
		maxRedeemable = s.BalanceMsat - dbTxFee
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"balance_msat": maxRedeemable,
		"expires_at":   s.ExpiresAt,
		"active":       s.Active,
		"refunded":     s.Refunded,
	})
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
	b, err := os.ReadFile("./static/redeem.html")
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	donateLNURL, _ := lnurlEncode(srv.cfg.baseURL + "/donate")
	html := strings.ReplaceAll(string(b), "{{BASE_URL}}", srv.cfg.baseURL)
	html = strings.ReplaceAll(html, "{{GITHUB_URL}}", srv.cfg.githubURL)
	html = strings.ReplaceAll(html, "{{DONATE_LNURL}}", donateLNURL)
	html = strings.ReplaceAll(html, "{{SITE_NAME_FULL}}", srv.cfg.siteName)
	html = strings.ReplaceAll(html, "{{SITE_LOGO_INNER}}", srv.cfg.siteLogoInner)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(html))
}

func (srv *Server) handleIndexPage(w http.ResponseWriter, r *http.Request) {
	b, err := os.ReadFile("./static/index.html")
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	donateLNURL, _ := lnurlEncode(srv.cfg.baseURL + "/donate")
	html := strings.ReplaceAll(string(b), "{{BASE_URL}}", srv.cfg.baseURL)
	html = strings.ReplaceAll(html, "{{GITHUB_URL}}", srv.cfg.githubURL)
	html = strings.ReplaceAll(html, "{{DONATE_LNURL}}", donateLNURL)
	html = strings.ReplaceAll(html, "{{SITE_NAME_FULL}}", srv.cfg.siteName)
	html = strings.ReplaceAll(html, "{{SITE_LOGO_INNER}}", srv.cfg.siteLogoInner)
	html = strings.ReplaceAll(html, "{{BATCH_ENABLED}}", strconv.FormatBool(srv.cfg.batchEnabled))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(html))
}

// GET /donate — LNURL-pay step 1
func (srv *Server) handleDonate(w http.ResponseWriter, r *http.Request) {
	if !srv.cfg.fundActive {
		lnurlError(w, "donations not enabled")
		return
	}
	metadata := [][]string{{"text/plain", "Donate to " + srv.cfg.siteName}}
	metaJSON, _ := json.Marshal(metadata)
	writeJSON(w, http.StatusOK, map[string]any{
		"tag":            "payRequest",
		"callback":       srv.cfg.baseURL + "/donate/callback",
		"minSendable":    srv.cfg.minFundAmountMsat,
		"maxSendable":    srv.cfg.maxFundAmountMsat,
		"metadata":       string(metaJSON),
		"commentAllowed": 200,
	})
}

// GET /donate/callback?amount=MSATS — LNURL-pay step 2
func (srv *Server) handleDonateCallback(w http.ResponseWriter, r *http.Request) {
	if !srv.cfg.fundActive {
		lnurlError(w, "donations not enabled")
		return
	}

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

	key, err := srv.insertDonation(pr, amountMsat, feeMsat, comment)
	if err != nil {
		slog.Error("insert donation", "err", err)
		lnurlError(w, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status": "OK",
		"pr":     pr,
		"routes": []any{},
		"verify": srv.cfg.baseURL + "/donate/verify/" + key,
	})
}

func (srv *Server) handleDonateVerify(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
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

func (srv *Server) handleAdminStats(w http.ResponseWriter, r *http.Request) {
	if srv.cfg.adminToken == "" || r.Header.Get("Authorization") != "Bearer "+srv.cfg.adminToken {
		w.WriteHeader(http.StatusUnauthorized)
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

func (srv *Server) handleAdminPage(w http.ResponseWriter, r *http.Request) {
	b, err := os.ReadFile("./static/admin.html")
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	html := strings.ReplaceAll(string(b), "{{SITE_NAME_FULL}}", srv.cfg.siteName)
	html = strings.ReplaceAll(html, "{{SITE_LOGO_INNER}}", srv.cfg.siteLogoInner)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(html))
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

	s.TotalDonations, s.ConfirmedDonations, s.DonatedMsat, err = srv.getDonationStats()
	if err != nil {
		return nil, err
	}

	return s, nil
}
