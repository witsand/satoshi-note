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

func (srv *Server) createVoucher(refundCode, batchName, batchID string, expiresAfterSeconds int64, singleUse bool) (*Voucher, error) {
	v := srv.newVoucher("", refundCode, batchName, batchID, expiresAfterSeconds, singleUse)

	if err := srv.insertVoucher(v); err != nil {
		return nil, err
	}

	return v, nil
}

func (srv *Server) newVoucher(pubKey, refundCode, batchName, batchID string, refundAfterSeconds int64, singleUse bool) *Voucher {
	return &Voucher{
		PubKey:               pubKey,
		FundURLPrefix:        srv.cfg.baseURL + "/f/",
		BatchName:            batchName,
		BatchID:              batchID,
		WithdrawURLPrefix:    srv.cfg.baseURL + "/w/",
		RefundCode:           refundCode,
		RefundAfterSeconds:   refundAfterSeconds,
		SingleUse:            singleUse,
		Active:               true,
	}
}
