package main

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
)

// calculateRedeemFee returns the fee in msat for a redeem, rounded down to the
// nearest sat and floored at the configured minimum.
func (srv *Server) calculateRedeemFee(balanceMsat int64) int64 {
	fee := balanceMsat*srv.cfg.redeemFeeBPS/10000/1000*1000 + 1000
	if fee < srv.cfg.minRedeemFeeMsat {
		fee = srv.cfg.minRedeemFeeMsat / 1000 * 1000
	}
	return fee
}

// calculateInternalFee returns the fee in msat for an internal wallet transfer,
// rounded down to the nearest sat and floored at the configured minimum.
func (srv *Server) calculateInternalFee(amountMsat int64) int64 {
	fee := amountMsat * srv.cfg.internalFeeBPS / 10000 / 1000 * 1000
	if fee < srv.cfg.minInternalFeeMsat {
		fee = srv.cfg.minInternalFeeMsat / 1000 * 1000
	}
	return fee
}

// voucherStatusBody builds the JSON body for a voucher status response.
func (srv *Server) voucherStatusBody(s *voucherStatus) map[string]any {
	redeemFee := srv.calculateRedeemFee(s.BalanceMsat)
	var maxRedeemable int64
	if s.BalanceMsat > redeemFee {
		maxRedeemable = s.BalanceMsat - redeemFee
	}
	return map[string]any{
		"balance_msat":     maxRedeemable,
		"raw_balance_msat": s.BalanceMsat,
		"expires_at":       s.ExpiresAt,
		"active":           s.Active,
		"refunded":         s.Refunded,
		"refund_pending":   s.RefundPending,
	}
}


// requireAdmin returns true if the request carries a valid admin token.
// It writes a 401 and returns false otherwise.
func (srv *Server) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	if srv.cfg.adminToken == "" || subtle.ConstantTimeCompare(
		[]byte(r.Header.Get("Authorization")),
		[]byte("Bearer "+srv.cfg.adminToken),
	) != 1 {
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}
	return true
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
