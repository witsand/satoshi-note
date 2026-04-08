package main

import (
	"encoding/json"
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
