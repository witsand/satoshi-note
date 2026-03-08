package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func (s *Server) createVoucher(refundCode string, expiresAfterSeconds int, singleWithdrawal bool) (*Voucher, error) {
	voucher, err := newVoucher()
	if err != nil {
		return nil, err
	}

	voucher.RefundCode = refundCode
	voucher.RefundAfterSeconds = expiresAfterSeconds
	if voucher.RefundAfterSeconds == 0 {
		voucher.RefundAfterSeconds = s.cfg.defaultRefundAfterSeconds
	}
	if voucher.RefundAfterSeconds > s.cfg.maxRefundAfterSeconds {
		voucher.RefundAfterSeconds = s.cfg.maxRefundAfterSeconds
	}
	voucher.SingleWithdrawal = singleWithdrawal
	voucher.Active = true

	if err := insertVoucher(s.db, voucher); err != nil {
		return nil, err
	}

	return voucher, nil
}

func newVoucher() (*Voucher, error) {
	secret := make([]byte, 16)
	if _, err := rand.Read(secret); err != nil {
		return nil, fmt.Errorf("generate secret: %w", err)
	}
	hash := sha256.Sum256(secret)
	return &Voucher{
		Secret: hex.EncodeToString(secret),
		PubKey: hex.EncodeToString(hash[:16]),
	}, nil
}
