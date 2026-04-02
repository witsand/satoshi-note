package main

import (
	"crypto/sha256"
	"encoding/hex"
)

// secretToPubKey re-derives the pubKey from a hex-encoded secret.
// secret is hex(randomBytes); pubKey is hex(sha256(randomBytes)[:len(randomBytes)]).
func secretToPubKey(secret string) (string, error) {
	b, err := hex.DecodeString(secret)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:len(b)]), nil
}

func (srv *Server) newVoucher(pubKey, refundCode, batchID string, refundAfterSeconds int64, singleUse, transfersOnly bool, maxRedeemMsat int64, uniqueRedemptions bool, absoluteExpiry bool) *Voucher {
	return &Voucher{
		PubKey:             pubKey,
		FundURLPrefix:      srv.cfg.baseURL + "/f/",
		BatchID:            batchID,
		WithdrawURLPrefix:  srv.cfg.baseURL + "/w/",
		RefundCode:         refundCode,
		RefundAfterSeconds: refundAfterSeconds,
		SingleUse:          singleUse,
		TransfersOnly:      transfersOnly,
		MaxRedeemMsat:      maxRedeemMsat,
		UniqueRedemptions:  uniqueRedemptions,
		AbsoluteExpiry:     absoluteExpiry,
		Active:             true,
	}
}
