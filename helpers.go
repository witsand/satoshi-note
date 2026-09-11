package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// satoshiNoteUUIDNamespace is the fixed RFC 4122 namespace UUID from which SDK
// payment idempotency keys are derived (UUIDv5, SHA-1). The value itself is
// arbitrary; only stability matters — the key for a given tx must be identical
// across restarts so the SDK can dedupe a retried payment.
const satoshiNoteUUIDNamespace = "3f5f1c2e-7a4b-4c1d-9e2f-5a6b7c8d9e0f"

// uuid5 returns the RFC 4122 UUIDv5 (SHA-1) of name within the given namespace
// UUID. Hand-rolled to avoid a new dependency for one small function. Panics on a
// malformed namespace — a programmer error that should fail fast at first use.
func uuid5(namespace, name string) string {
	hexPart := make([]byte, 0, 32)
	for i := 0; i < len(namespace); i++ {
		if namespace[i] != '-' {
			hexPart = append(hexPart, namespace[i])
		}
	}
	var ns [16]byte
	if _, err := hex.Decode(ns[:], hexPart); err != nil {
		panic(fmt.Sprintf("invalid namespace UUID %q: %v", namespace, err))
	}

	h := sha1.New()
	h.Write(ns[:])
	h.Write([]byte(name))
	sum := h.Sum(nil)

	var b [16]byte
	copy(b[:], sum[:16])
	b[6] = (b[6] & 0x0f) | 0x50 // version 5
	b[8] = (b[8] & 0x3f) | 0x80 // RFC 4122 variant
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// calculateRedeemFee returns the fee in msat for a redeem. The fee is the
// greater of the configured minimum fee and the bps fee computed on the net
// send amount (i.e. fee = balance - floor(balance / (1 + bps/10000))).
func (srv *Server) calculateRedeemFee(balanceMsat int64) int64 {
	netSat := balanceMsat * 10000 / (10000 + srv.cfg.redeemFeeBPS) / 1000
	bpsFee := balanceMsat - netSat*1000
	minFee := srv.cfg.minRedeemFeeMsat / 1000 * 1000
	if bpsFee > minFee {
		return bpsFee
	}
	return minFee
}

// calculateInternalFee returns the fee in msat for an internal wallet transfer.
// The fee is the greater of the configured minimum fee and the bps fee computed
// on the net send amount (i.e. fee = amount - floor(amount / (1 + bps/10000))).
func (srv *Server) calculateInternalFee(amountMsat int64) int64 {
	netSat := amountMsat * 10000 / (10000 + srv.cfg.internalFeeBPS) / 1000
	bpsFee := amountMsat - netSat*1000
	minFee := srv.cfg.minInternalFeeMsat / 1000 * 1000
	if bpsFee > minFee {
		return bpsFee
	}
	return minFee
}

// voucherStatusBody builds the JSON body for a voucher status response.
func (srv *Server) voucherStatusBody(s *voucherStatus) map[string]any {
	rawBalance := s.BalanceMsat
	if s.MaxRedeemMsat > 0 && s.MaxRedeemMsat < rawBalance {
		rawBalance = s.MaxRedeemMsat
	}

	body := map[string]any{
		"raw_balance_msat": rawBalance,
		"expires_at":       s.ExpiresAt,
		"active":           s.Active && !s.Expired,
		"expired":          s.Expired,
		"refunded":         s.Refunded,
		"refund_pending":   s.RefundPending,
		"last_refund_at":   s.LastRefundAt,
	}

	if !s.TransfersOnly {
		redeemFee := srv.calculateRedeemFee(rawBalance)
		var maxRedeemable int64
		if rawBalance > redeemFee {
			maxRedeemable = rawBalance - redeemFee
		}
		body["balance_msat"] = maxRedeemable
	}

	return body
}

// lnurlPayResponse builds a LNURL payRequest response map.
// extra keys (e.g. "commentAllowed") are merged in if provided.
func lnurlPayResponse(description, callback string, minMsat, maxMsat int64, extra map[string]any) map[string]any {
	meta, _ := json.Marshal([][]string{{"text/plain", description}})
	m := map[string]any{
		"tag":         "payRequest",
		"callback":    callback,
		"minSendable": minMsat,
		"maxSendable": maxMsat,
		"metadata":    string(meta),
	}
	for k, v := range extra {
		m[k] = v
	}
	return m
}
