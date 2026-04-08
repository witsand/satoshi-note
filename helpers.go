package main

import (
	"encoding/json"
)

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
