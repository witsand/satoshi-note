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
	"strconv"
	"strings"
	"time"

	spark "github.com/breez/breez-sdk-spark-go/breez_sdk_spark"
)

// voucherParams holds the shared fields present in both the create and edit request bodies.
type voucherParams struct {
	RefundCode  string `json:"refund_code"`
	RefundCodes []struct {
		Code  string `json:"code"`
		Share int64  `json:"share"`
	} `json:"refund_codes"`
	RefundAfterSeconds int64 `json:"refund_after_seconds"`
	SingleUse          bool  `json:"single_use"`
	TransfersOnly      bool  `json:"transfers_only"`
	MaxRedeemMsat      int64 `json:"max_redeem_msat"`
	UniqueRedemptions  bool  `json:"unique_redemptions"`
	AbsoluteExpiry     bool  `json:"absolute_expiry"`
	RegularRefund      *struct {
		FirstRefundAt   int64 `json:"first_refund_at"`
		IntervalSeconds int64 `json:"interval_seconds"`
	} `json:"regular_refund"`
}

type parsedVoucherParams struct {
	NormalizedCodes     []VoucherRefundCode
	RefundAfterSeconds  int64
	SingleUse           bool
	TransfersOnly       bool
	MaxRedeemMsat       int64
	UniqueRedemptions   bool
	AbsoluteExpiry      bool
	RegularFirstAt      int64
	RegularIntervalSecs int64
}

// parseVoucherParams validates and normalizes the shared voucher fields.
// Returns (parsed, 0, "") on success, or (zero, httpStatus, errMsg) on failure.
func (srv *Server) parseVoucherParams(p voucherParams) (parsedVoucherParams, int, string) {
	var out parsedVoucherParams

	if p.RefundCode != "" && len(p.RefundCodes) > 0 {
		return out, http.StatusUnprocessableEntity, "provide refund_code or refund_codes, not both"
	} else if p.RefundCode != "" {
		out.NormalizedCodes = []VoucherRefundCode{{RefundCode: strings.ToLower(p.RefundCode), Share: 1}}
	} else if len(p.RefundCodes) > 0 {
		for _, rc := range p.RefundCodes {
			if rc.Code == "" {
				return out, http.StatusBadRequest, "each refund_codes entry must have a non-empty code"
			}
			if rc.Share <= 0 {
				return out, http.StatusBadRequest, "each refund_codes entry must have a share greater than 0"
			}
			out.NormalizedCodes = append(out.NormalizedCodes, VoucherRefundCode{RefundCode: strings.ToLower(rc.Code), Share: rc.Share})
		}
	} else {
		return out, http.StatusBadRequest, "refund_code or refund_codes is required"
	}

	if p.RefundAfterSeconds <= 0 {
		return out, http.StatusBadRequest, "refund_after_seconds must be greater than 0"
	}
	out.RefundAfterSeconds = p.RefundAfterSeconds
	if out.RefundAfterSeconds > srv.cfg.maxVoucherExpireSeconds {
		out.RefundAfterSeconds = srv.cfg.maxVoucherExpireSeconds
	}

	if p.MaxRedeemMsat > 0 && p.SingleUse {
		return out, http.StatusUnprocessableEntity, "max_redeem_msat cannot be set on a single_use voucher"
	}
	if p.UniqueRedemptions && p.SingleUse {
		return out, http.StatusUnprocessableEntity, "single_use cannot be set on a unique_redemptions voucher"
	}
	if p.UniqueRedemptions && !p.TransfersOnly {
		return out, http.StatusUnprocessableEntity, "unique_redemptions vouchers must be transfers_only"
	}
	out.SingleUse = p.SingleUse
	out.TransfersOnly = p.TransfersOnly
	out.MaxRedeemMsat = p.MaxRedeemMsat
	out.UniqueRedemptions = p.UniqueRedemptions
	out.AbsoluteExpiry = p.AbsoluteExpiry

	if p.RegularRefund != nil {
		if p.RegularRefund.FirstRefundAt <= 0 {
			return out, http.StatusBadRequest, "regular_refund.first_refund_at must be a valid unix timestamp"
		}
		if p.RegularRefund.IntervalSeconds <= 0 {
			return out, http.StatusBadRequest, "regular_refund.interval_seconds must be greater than 0"
		}
		out.RegularFirstAt = p.RegularRefund.FirstRefundAt
		out.RegularIntervalSecs = p.RegularRefund.IntervalSeconds
	}

	return out, 0, ""
}

func lnurlError(w http.ResponseWriter, status int, reason string) {
	writeJSON(w, status, map[string]string{
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
		lnurlError(w, http.StatusServiceUnavailable, "creating is currently disabled")
		return
	}

	var req struct {
		PubKeys []string `json:"pub_keys"`
		voucherParams
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		slog.Error("decode request body", "err", err)
		lnurlError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.PubKeys) == 0 {
		lnurlError(w, http.StatusBadRequest, "pub_keys must not be empty")
		return
	}
	if int64(len(req.PubKeys)) > srv.cfg.maxVouchersPerBatch {
		lnurlError(w, http.StatusBadRequest, "too many vouchers")
		return
	}
	for _, pk := range req.PubKeys {
		b, err := hex.DecodeString(pk)
		if err != nil || len(b) < 16 || len(b) > 32 {
			lnurlError(w, http.StatusBadRequest, "invalid pub_key: must be hex, 16–32 bytes")
			return
		}
	}

	p, status, errMsg := srv.parseVoucherParams(req.voucherParams)
	if status != 0 {
		lnurlError(w, status, errMsg)
		return
	}

	pubKeyLen := len(req.PubKeys[0]) / 2
	batchIDBytes := make([]byte, pubKeyLen)
	if _, err := rand.Read(batchIDBytes); err != nil {
		slog.Error("create batch id error", "err", err)
		lnurlError(w, http.StatusInternalServerError, "internal error")
		return
	}
	batchID := hex.EncodeToString(batchIDBytes)

	// Create all vouchers in a single DB transaction so a partial failure leaves no orphaned rows.
	dbTx, err := srv.db.Begin()
	if err != nil {
		slog.Error("begin transaction", "err", err)
		lnurlError(w, http.StatusInternalServerError, "internal error")
		return
	}
	defer dbTx.Rollback()

	var vs []Voucher
	for _, pubKey := range req.PubKeys {
		voucher := srv.newVoucher(pubKey, batchID, p.RefundAfterSeconds, p.SingleUse, p.TransfersOnly, p.MaxRedeemMsat, p.UniqueRedemptions, p.AbsoluteExpiry, p.RegularFirstAt, p.RegularIntervalSecs)
		// fund_key is a one-way SHA256 derivation from pub_key; safe to ignore error since pub_key is already validated hex.
		voucher.FundKey, _ = secretToPubKey(pubKey)

		result, err := dbTx.Exec(
			`INSERT INTO vouchers (pub_key, fund_key, batch_id, refund_after_seconds, single_use, transfers_only, max_redeem_msat, unique_redemptions, absolute_expiry, regular_refund_first_at, regular_refund_interval_secs, regular_refund_last_at, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?)`,
			voucher.PubKey, voucher.FundKey, voucher.BatchID,
			voucher.RefundAfterSeconds, boolToInt(voucher.SingleUse),
			boolToInt(voucher.TransfersOnly), voucher.MaxRedeemMsat, boolToInt(voucher.UniqueRedemptions),
			boolToInt(voucher.AbsoluteExpiry), voucher.RegularRefundFirstAt, voucher.RegularRefundIntervalSecs,
			time.Now().Unix(),
		)
		if err != nil {
			slog.Error("insert voucher", "err", err)
			lnurlError(w, http.StatusInternalServerError, "internal error")
			return
		}

		voucherID, err := result.LastInsertId()
		if err != nil {
			slog.Error("get voucher id", "err", err)
			lnurlError(w, http.StatusInternalServerError, "internal error")
			return
		}

		if err := insertVoucherRefundCodes(dbTx, voucherID, p.NormalizedCodes); err != nil {
			slog.Error("insert voucher refund codes", "err", err)
			lnurlError(w, http.StatusInternalServerError, "internal error")
			return
		}

		vs = append(vs, *voucher)
	}

	if err := dbTx.Commit(); err != nil {
		slog.Error("commit vouchers", "err", err)
		lnurlError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Signal the refund worker to re-evaluate its sleep in case these vouchers
	// have a sooner regular_refund or expiry than the current scheduled wake.
	select {
	case srv.refundWake <- struct{}{}:
	default:
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(vs); err != nil {
		slog.Error("encode response", "err", err)
	}
}

// POST /edit — update voucher configuration, authenticated by secret_secret
func (srv *Server) handleEditVoucher(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SecretSecret string `json:"secret_secret"`
		voucherParams
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		slog.Error("edit: decode request body", "err", err)
		lnurlError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate secret_secret and derive pub_key via double application of secretToPubKey.
	secretBytes, err := hex.DecodeString(req.SecretSecret)
	if err != nil || len(secretBytes) < 16 || len(secretBytes) > 32 {
		lnurlError(w, http.StatusBadRequest, "invalid secret_secret: must be hex, 16–32 bytes")
		return
	}
	secret, err := secretToPubKey(req.SecretSecret)
	if err != nil {
		lnurlError(w, http.StatusBadRequest, "invalid secret_secret")
		return
	}
	pubKey, err := secretToPubKey(secret)
	if err != nil {
		lnurlError(w, http.StatusBadRequest, "invalid secret_secret")
		return
	}

	p, status, errMsg := srv.parseVoucherParams(req.voucherParams)
	if status != 0 {
		lnurlError(w, status, errMsg)
		return
	}

	dbTx, err := srv.db.Begin()
	if err != nil {
		slog.Error("edit: begin transaction", "err", err)
		lnurlError(w, http.StatusInternalServerError, "internal error")
		return
	}
	defer dbTx.Rollback()

	voucherID, err := srv.getVoucherIDByPubKey(dbTx, pubKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			lnurlError(w, http.StatusNotFound, "voucher not found")
		} else {
			slog.Error("edit: get voucher id", "err", err)
			lnurlError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	if err := deleteVoucherRefundCodes(dbTx, voucherID); err != nil {
		slog.Error("edit: delete refund codes", "err", err)
		lnurlError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if err := insertVoucherRefundCodes(dbTx, voucherID, p.NormalizedCodes); err != nil {
		slog.Error("edit: insert refund codes", "err", err)
		lnurlError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if err := srv.updateVoucher(dbTx, voucherID, p.RefundAfterSeconds, p.SingleUse, p.TransfersOnly,
		p.MaxRedeemMsat, p.UniqueRedemptions, p.AbsoluteExpiry, p.RegularFirstAt, p.RegularIntervalSecs); err != nil {
		slog.Error("edit: update voucher", "err", err)
		lnurlError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if err := dbTx.Commit(); err != nil {
		slog.Error("edit: commit", "err", err)
		lnurlError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Signal refund worker to re-evaluate sleep in case the new schedule is sooner.
	select {
	case srv.refundWake <- struct{}{}:
	default:
	}

	v := srv.newVoucher(pubKey, "", p.RefundAfterSeconds, p.SingleUse, p.TransfersOnly,
		p.MaxRedeemMsat, p.UniqueRedemptions, p.AbsoluteExpiry, p.RegularFirstAt, p.RegularIntervalSecs)
	v.FundKey, _ = secretToPubKey(pubKey)
	v.Active = true

	writeJSON(w, http.StatusOK, v)
}

// GET /f/{pubKey} — LNURL-pay step 1
func (srv *Server) handleLNURLPayVoucher(w http.ResponseWriter, r *http.Request) {
	if !srv.cfg.fundActive {
		lnurlError(w, http.StatusOK, "funding is currently disabled")
		return
	}

	key := r.PathValue("pubKey")

	lookupSingle := func() (*Voucher, bool) {
		if v, err := srv.getVoucherByFundKey(srv.db, key); err == nil {
			return v, true
		}
		if v, err := srv.getVoucherByPubKey(srv.db, key); err == nil {
			return v, true
		}
		return nil, false
	}

	if v, ok := lookupSingle(); ok {
		remaining := srv.cfg.maxFundAmountMsat - v.BalanceMsat
		if remaining < 1000 {
			lnurlError(w, http.StatusOK, "voucher is fully funded")
			return
		}
		writeJSON(w, http.StatusOK, lnurlPayResponse(
			"Fund a Voucher",
			srv.cfg.baseURL+"/fund/"+key+"/callback",
			1000, remaining,
			nil,
		))
		return
	}

	vs, err := srv.getVouchersByBatchID(srv.db, key)
	if err != nil {
		lnurlError(w, http.StatusOK, "voucher or batch not found")
		return
	}

	n := int64(len(vs))
	minRemaining := srv.cfg.maxFundAmountMsat
	for _, v := range vs {
		if rem := srv.cfg.maxFundAmountMsat - v.BalanceMsat; rem < minRemaining {
			minRemaining = rem
		}
	}
	batchMax := minRemaining * n
	if batchMax < 1000*n {
		lnurlError(w, http.StatusOK, "batch vouchers are fully funded")
		return
	}
	writeJSON(w, http.StatusOK, lnurlPayResponse(
		"Fund a Batch Vouchers",
		srv.cfg.baseURL+"/fund/"+key+"/callback",
		1000*n, batchMax,
		nil,
	))
}

// GET /fund/{pubKey}/callback?amount=MSATS — LNURL-pay step 2
func (srv *Server) handleLNURLPayCallbackVoucher(w http.ResponseWriter, r *http.Request) {
	if !srv.cfg.fundActive {
		lnurlError(w, http.StatusOK, "funding is currently disabled")
		return
	}

	key := r.PathValue("pubKey")

	tx := &FundTx{}

	if v, err := func() (*Voucher, error) {
		if v, err := srv.getVoucherByFundKey(srv.db, key); err == nil {
			return v, nil
		}
		return srv.getVoucherByPubKey(srv.db, key)
	}(); err == nil {
		tx.PubKey = v.PubKey // always the real pub_key, not the path key
		var err error
		tx.Msat, err = srv.getCallbackAmount(r, 1)
		if err != nil {
			lnurlError(w, http.StatusOK, "invalid amount")
			return
		}
		if tx.Msat+v.BalanceMsat > srv.cfg.maxFundAmountMsat {
			lnurlError(w, http.StatusOK, "amount would exceed maximum voucher balance")
			return
		}
		if err = srv.getCallbackBolt11(tx, "Fund a Voucher"); err != nil {
			slog.Error("create invoice", "err", err)
			lnurlError(w, http.StatusOK, "failed to create invoice")
			return
		}
	} else {
		vs, err := srv.getVouchersByBatchID(srv.db, key)
		if err != nil {
			lnurlError(w, http.StatusOK, "voucher or batch not found")
			return
		}
		tx.BatchID = key
		n := int64(len(vs))
		tx.Msat, err = srv.getCallbackAmount(r, n)
		if err != nil {
			lnurlError(w, http.StatusOK, err.Error())
			return
		}
		minRemaining := srv.cfg.maxFundAmountMsat
		for _, v := range vs {
			if rem := srv.cfg.maxFundAmountMsat - v.BalanceMsat; rem < minRemaining {
				minRemaining = rem
			}
		}
		if tx.Msat > minRemaining*n {
			lnurlError(w, http.StatusOK, "amount would exceed maximum voucher balance")
			return
		}
		if err = srv.getCallbackBolt11(tx, "Fund a Batch Vouchers"); err != nil {
			slog.Error("create invoice", "err", err)
			lnurlError(w, http.StatusOK, "failed to create invoice")
			return
		}
	}

	if err := srv.insertFundTX(tx); err != nil {
		slog.Error("insert fund tx", "err", err)
		lnurlError(w, http.StatusOK, "failed to write fund tx")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status": "OK",
		"pr":     tx.PR,
		"routes": []any{},
		"verify": srv.cfg.baseURL + "/verify/" + tx.Key,
	})
}

// POST /transfer — move funds from a non-single-use voucher to any destination (pubKey or batchID)
func (srv *Server) handleTransfer(w http.ResponseWriter, r *http.Request) {
	if !srv.cfg.fundActive || !srv.cfg.redeemActive {
		lnurlError(w, http.StatusServiceUnavailable, "transfers are currently disabled")
		return
	}

	var req struct {
		Secret      string `json:"secret"`
		PubKey      string `json:"pub_key"`
		AmountMsat  int64  `json:"amount_msat"`
		Fingerprint string `json:"fingerprint"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		lnurlError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Secret == "" || req.PubKey == "" {
		lnurlError(w, http.StatusBadRequest, "secret and pub_key are required")
		return
	}

	// Resolve source voucher
	srcPubKey, err := secretToPubKey(req.Secret)
	if err != nil {
		lnurlError(w, http.StatusBadRequest, "invalid secret")
		return
	}
	src, err := srv.getVoucherByPubKey(srv.db, srcPubKey)
	if err != nil {
		lnurlError(w, http.StatusNotFound, "source voucher not found")
		return
	}
	if src.SingleUse {
		lnurlError(w, http.StatusUnprocessableEntity, "single-use vouchers cannot transfer funds")
		return
	}

	if src.UniqueRedemptions && req.Fingerprint == "" {
		lnurlError(w, http.StatusBadRequest, "fingerprint required for this voucher")
		return
	}
	if src.UniqueRedemptions && req.Fingerprint != "" {
		used, err := srv.usedFingerprints([]int64{src.ID}, req.Fingerprint)
		if err != nil {
			slog.Error("check fingerprint", "err", err)
			lnurlError(w, http.StatusInternalServerError, "internal error")
			return
		}
		if used[src.ID] {
			lnurlError(w, http.StatusConflict, "this fingerprint has already redeemed this voucher")
			return
		}
	}

	// Resolve destination
	var dstVoucher *Voucher
	var dstVouchers []Voucher
	var dstPubKey, dstBatchID string

	if v, err := srv.getVoucherByFundKey(srv.db, req.PubKey); err == nil {
		if v.PubKey == srcPubKey {
			lnurlError(w, http.StatusBadRequest, "source and destination cannot be the same voucher")
			return
		}
		dstVoucher = v
		dstPubKey = v.PubKey // real pub_key for the transfer record
	} else if v, err := srv.getVoucherByPubKey(srv.db, req.PubKey); err == nil {
		if req.PubKey == srcPubKey {
			lnurlError(w, http.StatusBadRequest, "source and destination cannot be the same voucher")
			return
		}
		dstVoucher = v
		dstPubKey = req.PubKey
	} else {
		vs, err := srv.getVouchersByBatchID(srv.db, req.PubKey)
		if err != nil {
			lnurlError(w, http.StatusNotFound, "destination not found")
			return
		}
		dstVouchers = vs
		dstBatchID = req.PubKey
	}

	dstCount := int64(1)
	if dstBatchID != "" {
		dstCount = int64(len(dstVouchers))
	}

	amountMsat := req.AmountMsat
	if amountMsat < 1000*dstCount || amountMsat > srv.cfg.maxFundAmountMsat*dstCount {
		lnurlError(w, http.StatusBadRequest, "amount_msat out of range")
		return
	}

	if amountMsat > src.BalanceMsat {
		lnurlError(w, http.StatusUnprocessableEntity, "insufficient balance")
		return
	}

	if src.MaxRedeemMsat > 0 && amountMsat > src.MaxRedeemMsat {
		lnurlError(w, http.StatusUnprocessableEntity, "amount exceeds per-transfer limit")
		return
	}

	// Calculate fee (rounded down to nearest sat)
	feeMsat := amountMsat * srv.cfg.internalFeeBPS / 10000 / 1000 * 1000
	if feeMsat < srv.cfg.minInternalFeeMsat {
		feeMsat = srv.cfg.minInternalFeeMsat / 1000 * 1000
	}
	netMsat := amountMsat - feeMsat

	// Execute atomically
	dbTx, err := srv.db.Begin()
	if err != nil {
		slog.Error("transfer begin tx", "err", err)
		lnurlError(w, http.StatusInternalServerError, "internal error")
		return
	}
	defer dbTx.Rollback()

	// Deduct from source
	if err := srv.updateVoucherBalance(dbTx, src.ID, -amountMsat); err != nil {
		lnurlError(w, http.StatusUnprocessableEntity, "insufficient balance")
		return
	}

	// Credit destination
	var dustMsat int64
	if dstVoucher != nil {
		if err := srv.updateVoucherBalance(dbTx, dstVoucher.ID, netMsat); err != nil {
			slog.Error("transfer credit voucher", "err", err)
			lnurlError(w, http.StatusInternalServerError, "failed to credit destination")
			return
		}
	} else {
		share := netMsat / int64(len(dstVouchers)) / 1000 * 1000
		dustMsat = netMsat - share*int64(len(dstVouchers))
		for _, v := range dstVouchers {
			if err := srv.updateVoucherBalance(dbTx, v.ID, share); err != nil {
				slog.Error("transfer credit batch voucher", "err", err)
				lnurlError(w, http.StatusInternalServerError, "failed to credit destination")
				return
			}
		}
	}

	if err := srv.insertTransferTx(dbTx, srcPubKey, dstPubKey, dstBatchID, amountMsat, feeMsat, netMsat, dustMsat); err != nil {
		slog.Error("insert transfer tx", "err", err)
		lnurlError(w, http.StatusInternalServerError, "failed to record transfer")
		return
	}

	if src.UniqueRedemptions {
		inserted, err := insertRedemptionFingerprint(dbTx, src.ID, req.Fingerprint)
		if err != nil {
			slog.Error("insert redemption fingerprint", "err", err)
			lnurlError(w, http.StatusInternalServerError, "internal error")
			return
		}
		if !inserted {
			lnurlError(w, http.StatusConflict, "this fingerprint has already redeemed this voucher")
			return
		}
	}

	if err := dbTx.Commit(); err != nil {
		slog.Error("transfer commit", "err", err)
		lnurlError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"amount":   amountMsat,
		"fee_msat": feeMsat,
		"net_msat": netMsat,
	})
}

func (srv *Server) handleLNURLVerify(w http.ResponseWriter, r *http.Request) {
	if !srv.cfg.fundActive {
		lnurlError(w, http.StatusOK, "funding is currently disabled")
		return
	}

	key := r.PathValue("key")

	tx, err := srv.getFundTxByKey(key)
	if err != nil {
		lnurlError(w, http.StatusOK, "not found")
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
		lnurlError(w, http.StatusOK, "redeem is currently disabled")
		return
	}

	secret := r.PathValue("secret")

	pubKey, err := secretToPubKey(secret)
	if err != nil {
		lnurlError(w, http.StatusOK, "invalid secret")
		return
	}

	v, err := srv.getVoucherByPubKey(srv.db, pubKey)
	if err != nil {
		lnurlError(w, http.StatusOK, "voucher not found")
		return
	}

	if v.TransfersOnly {
		lnurlError(w, http.StatusOK, "this voucher can only be transferred, not redeemed")
		return
	}

	k1Bytes := make([]byte, len(pubKey)/2)
	if _, err := rand.Read(k1Bytes); err != nil {
		lnurlError(w, http.StatusOK, "internal error")
		return
	}
	k1 := hex.EncodeToString(k1Bytes)

	err = srv.insertRedeemSession(k1, pubKey)
	if err != nil {
		slog.Error("insert redeem session", "err", err)
		lnurlError(w, http.StatusOK, "internal error")
		return
	}

	dbTxFee := srv.calculateRedeemFee(v.BalanceMsat)

	maxRedeemable := v.BalanceMsat - dbTxFee
	if v.MaxRedeemMsat > 0 && v.MaxRedeemMsat < maxRedeemable {
		maxRedeemable = v.MaxRedeemMsat
	}
	minRedeemable := int64(1000)

	if minRedeemable > maxRedeemable {
		lnurlError(w, http.StatusOK, "voucher balance too low")
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
		"defaultDescription": "Redeem Voucher",
	})
}

func (srv *Server) handleLNURLWithdrawCallback(w http.ResponseWriter, r *http.Request) {
	if !srv.cfg.redeemActive {
		lnurlError(w, http.StatusOK, "redeem is currently disabled")
		return
	}

	secret := r.PathValue("secret")

	pubKey, err := secretToPubKey(secret)
	if err != nil {
		lnurlError(w, http.StatusOK, "invalid secret")
		return
	}

	k1 := r.URL.Query().Get("k1")
	if k1 == "" {
		lnurlError(w, http.StatusOK, "missing k1")
		return
	}

	pr := r.URL.Query().Get("pr")
	if pr == "" {
		lnurlError(w, http.StatusOK, "missing pr")
		return
	}

	if !srv.paymentSema.acquireForWithdrawal() {
		lnurlError(w, http.StatusOK, "server busy, please retry")
		return
	}
	defer srv.paymentSema.releaseAfter(srv.cfg.paymentCooldown)

	err = srv.markRedeemSessionUsed(k1, pubKey)
	if err != nil {
		lnurlError(w, http.StatusOK, "invalid or expired k1")
		return
	}

	prepResp, rawPrepErr := srv.ln.PrepareSendPayment(spark.PrepareSendPaymentRequest{
		PaymentRequest: pr,
	})
	if err := sdkErr(rawPrepErr); err != nil {
		slog.Error("prepare send payment", "err", err)
		lnurlError(w, http.StatusOK, "prepare payment failed")
		return
	}

	var estimateFeeMsat int64
	var amountMsat int64
	switch pm := prepResp.PaymentMethod.(type) {
	case spark.SendPaymentMethodBolt11Invoice:
		if pm.InvoiceDetails.AmountMsat == nil {
			lnurlError(w, http.StatusOK, "zero-amount invoices are not supported")
			return
		}
		estimateFeeMsat = int64(pm.LightningFeeSats) * 1000
		amountMsat = int64(*pm.InvoiceDetails.AmountMsat)
	default:
		slog.Error("unsupported payment method", "type", fmt.Sprintf("%T", prepResp.PaymentMethod))
		lnurlError(w, http.StatusOK, "unsupported payment method")
		return
	}

	v, err := srv.getVoucherByPubKey(srv.db, pubKey)
	if err != nil {
		lnurlError(w, http.StatusOK, "voucher not found")
		return
	}

	if v.TransfersOnly {
		lnurlError(w, http.StatusOK, "this voucher can only be transferred, not redeemed")
		return
	}

	if v.MaxRedeemMsat > 0 && amountMsat > v.MaxRedeemMsat {
		lnurlError(w, http.StatusOK, "redeem amount exceeds per-redeem limit")
		return
	}

	// Balance deduction, fee calculation, and redeem_tx insertion are all performed
	// atomically inside insertRedeemAndBalance. The fee is computed from the actual
	// current balance (re-read inside the transaction) so stale outer reads can't
	// cause an over- or under-deduction.
	redeemID, dbTxFee, err := srv.insertRedeemAndBalance(v, secret, pr, amountMsat)
	if err != nil {
		slog.Error("insert redeem and balance", "err", err)
		lnurlError(w, http.StatusOK, "redeem failed: "+err.Error())
		return
	}

	// Routing-fee check must happen after the deduction since the service fee
	// is computed from the actual balance. Undo the deduction if the check fails.
	if estimateFeeMsat > dbTxFee {
		if restoreErr := srv.addVoucherBalance(v.ID, amountMsat+dbTxFee); restoreErr != nil {
			slog.Error("restore voucher balance after routing fee rejection", "voucher_id", v.ID, "err", restoreErr)
		}
		if markErr := srv.updateRedeemTx(redeemID, TxFailed, 0, 0, "routing fee too high"); markErr != nil {
			slog.Error("mark redeem tx failed (routing fee)", "id", redeemID, "err", markErr)
		}
		lnurlError(w, http.StatusOK, "routing fee too high")
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
			slog.Error("restore voucher balance after failed payment", "voucher_id", v.ID, "err", err)
		}
		lnurlError(w, http.StatusOK, "payment failed")
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

// insertRedeemAndBalance atomically deducts the voucher balance and records the
// redeem_tx as TxPending. The service fee is computed from the balance re-read
// inside the transaction so concurrent operations between the outer voucher lookup
// and this call cannot cause a stale fee or an incorrect amount check.
// Returns (redeemID, actualDbTxFee, error).
func (srv *Server) insertRedeemAndBalance(v *Voucher, secret, pr string, amountMsat int64) (int64, int64, error) {
	dbTx, err := srv.db.Begin()
	if err != nil {
		return 0, 0, err
	}
	defer dbTx.Rollback()

	// Re-read balance inside the transaction to get the authoritative value.
	var currentBalance int64
	if err := dbTx.QueryRow(
		`SELECT balance_msat FROM vouchers WHERE id = ? AND active = 1`, v.ID,
	).Scan(&currentBalance); err != nil {
		return 0, 0, fmt.Errorf("voucher not found or inactive")
	}

	dbTxFee := srv.calculateRedeemFee(currentBalance)

	if amountMsat+dbTxFee > currentBalance {
		return 0, 0, fmt.Errorf("insufficient balance after fee")
	}

	if err := srv.updateVoucherBalance(dbTx, v.ID, -amountMsat-dbTxFee); err != nil {
		return 0, 0, err
	}

	redeemID, err := srv.insertRedeemTx(dbTx, v.ID, secret, pr, amountMsat, dbTxFee)
	if err != nil {
		return 0, 0, err
	}

	return redeemID, dbTxFee, dbTx.Commit()
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

func (srv *Server) handleVoucherStatusBatch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PubKeys     []string `json:"pubkeys"`
		Fingerprint string   `json:"fingerprint"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		lnurlError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if len(req.PubKeys) > 500 {
		lnurlError(w, http.StatusBadRequest, "too many pubkeys")
		return
	}

	statuses, err := srv.getVoucherStatusBatch(req.PubKeys)
	if err != nil {
		lnurlError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Collect IDs of unique_redemptions vouchers so we can check fingerprint usage in one query.
	var uniqueIDs []int64
	var usedIDs map[int64]bool
	if req.Fingerprint != "" {
		for _, s := range statuses {
			if s.UniqueRedemptions {
				uniqueIDs = append(uniqueIDs, s.ID)
			}
		}
		if len(uniqueIDs) > 0 && req.Fingerprint != "" {
			usedIDs, err = srv.usedFingerprints(uniqueIDs, req.Fingerprint)
			if err != nil {
				lnurlError(w, http.StatusInternalServerError, "internal error")
				return
			}
		}
	}

	result := make(map[string]any, len(req.PubKeys))
	for _, pubKey := range req.PubKeys {
		s, ok := statuses[pubKey]
		if !ok {
			continue
		}
		if s.UniqueRedemptions && req.Fingerprint == "" {
			continue
		}
		if s.UniqueRedemptions && usedIDs[s.ID] {
			s.BalanceMsat = 0
			s.Active = false
		}
		result[pubKey] = srv.voucherStatusBody(s)
	}

	writeJSON(w, http.StatusOK, result)
}

func (srv *Server) getCallbackAmount(r *http.Request, n int64) (int64, error) {
	msatsStr := r.URL.Query().Get("amount")

	if msatsStr == "" {
		return 0, fmt.Errorf("missing amount")
	}
	msats, err := strconv.ParseInt(msatsStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount")
	}
	if msats < 1000*n || msats > srv.cfg.maxFundAmountMsat*n {
		return 0, fmt.Errorf("amount out of range")
	}
	return msats, nil
}

func (srv *Server) handleLedger(w http.ResponseWriter, r *http.Request) {
	infoResp, err := srv.ln.GetInfo(spark.GetInfoRequest{})
	if err := sdkErr(err); err != nil {
		slog.Error("get sdk info", "err", err)
		lnurlError(w, http.StatusInternalServerError, "internal error")
		return
	}

	stats, err := srv.getLedgerStats()
	if err != nil {
		slog.Error("get ledger stats", "err", err)
		lnurlError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"sdk_balance_msat":             int64(infoResp.BalanceSats) * 1000,
		"vouchers_balance_msat":        stats.VouchersBalanceMsat,
		"fund_txs_dust_msat":           stats.FundTxsDustMsat,
		"refund_txs_db_tx_fee":         stats.RefundTxsDbTxFee,
		"refund_txs_pending_msat":      stats.RefundTxsPendingMsat,
		"redeem_txs_db_tx_fee":         stats.RedeemTxsDbTxFee,
		"transfer_txs_fee_msat":        stats.TransferTxsFeeMsat,
		"transfer_txs_dust_msat":       stats.TransferTxsDustMsat,
		"health":                       int64(infoResp.BalanceSats)*1000 - stats.VouchersBalanceMsat - stats.RefundTxsPendingMsat,
		"vouchers_avg_hours_to_expiry": stats.VouchersAvgSecsToExpiry / 60 / 60,
		"vouchers_with_balance_count":  stats.VouchersWithBalanceCount,
	})
}

func (srv *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"min_fund_amount_msat": int64(1000),
		"base_url":             srv.cfg.baseURL,
	})
}
