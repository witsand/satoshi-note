package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
)

type Server struct {
	db  *sql.DB
	cfg *Config
}

func (s *Server) handleCreateVoucher(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefundAddress      string `json:"refund_address"`
		RefundAfterSeconds int    `json:"refund_after_seconds"`
		SingleWithdrawal   bool   `json:"single_withdrawal"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		slog.Error("decode request body", "err", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	voucher, err := s.createVoucher(req.RefundAddress, req.RefundAfterSeconds, req.SingleWithdrawal)
	if err != nil {
		slog.Error("create voucher", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(voucher); err != nil {
		slog.Error("encode response", "err", err)
	}
}

func (s *Server) handleCreateVouchers(w http.ResponseWriter, r *http.Request) {
	sAmount := r.PathValue("amount")
	amount, err := strconv.Atoi(sAmount)
	if err != nil || amount <= 0 {
		http.Error(w, "invalid amount", http.StatusBadRequest)
		return
	}

	if amount > s.cfg.maxVouchers {
		http.Error(w, "too many vouchers requested", http.StatusBadRequest)
		return
	}

	var req struct {
		RefundCode          string `json:"refund_code"`
		ExpiresAfterSeconds int    `json:"expires_after_seconds"`
		SingleWithdrawal    bool   `json:"single_withdrawal"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		slog.Error("decode request body", "err", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var vs []Voucher
	for i := 0; i < amount; i++ {
		voucher, err := s.createVoucher(req.RefundCode, req.ExpiresAfterSeconds, req.SingleWithdrawal)
		if err != nil {
			slog.Error("create voucher", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		vs = append(vs, *voucher)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(vs); err != nil {
		slog.Error("encode response", "err", err)
	}
}
