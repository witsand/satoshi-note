package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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
	v, err := srv.newVoucher(refundCode, batchName, batchID, expiresAfterSeconds, singleUse)
	if err != nil {
		return nil, err
	}

	if err := srv.insertVoucher(v); err != nil {
		return nil, err
	}

	return v, nil
}

func (srv *Server) newVoucher(refundCode, batchName, batchID string, refundAfterSeconds int64, singleUse bool) (*Voucher, error) {
	secretBytes := make([]byte, srv.cfg.randomBytesLength)
	if _, err := rand.Read(secretBytes); err != nil {
		return nil, fmt.Errorf("generate secret: %w", err)
	}

	secretHash := sha256.Sum256(secretBytes)
	secret := hex.EncodeToString(secretBytes[:srv.cfg.randomBytesLength])
	// Use the full 32-byte SHA256 hash as the public key to avoid truncation issues.
	pubKey := hex.EncodeToString(secretHash[:srv.cfg.randomBytesLength])

	claimLNURL, err := lnurlEncode(srv.cfg.baseURL + "/w/" + secret)
	if err != nil {
		return nil, fmt.Errorf("encode claim LNURL: %w", err)
	}

	fundLNURL, err := lnurlEncode(srv.cfg.baseURL + "/f/" + pubKey)
	if err != nil {
		return nil, fmt.Errorf("encode fund LNURL: %w", err)
	}

	batchFundLNURL, err := lnurlEncode(srv.cfg.baseURL + "/fb/" + batchID)
	if err != nil {
		return nil, fmt.Errorf("encode batch fund LNURL: %w", err)
	}

	return &Voucher{
		Secret:             secret,
		ClaimLNURL:         claimLNURL,
		PubKey:             pubKey,
		FundLNURL:          fundLNURL,
		BatchName:          batchName,
		BatchID:            batchID,
		BatchFundLNURL:     batchFundLNURL,
		RefundCode:         refundCode,
		RefundAfterSeconds: refundAfterSeconds,
		SingleUse:          singleUse,
		Active:             true,
	}, nil
}
