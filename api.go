package main

import (
	"crypto/subtle"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (srv *Server) ServeAPI() {
	strict := newRateLimiter(rate.Every(20*time.Second), 2).Middleware
	api := newRateLimiter(rate.Every(5*time.Second), 50).Middleware
	lnurl := newRateLimiter(rate.Every(5*time.Second), 8).Middleware

	mux := http.NewServeMux()

	// Admin UI (static page; the APIs it calls are bearer-gated)
	mux.Handle("GET /{$}", api(http.HandlerFunc(handleAdminUI)))

	// Strict (1 req/2s, burst 5)
	mux.Handle("POST /create", strict(http.HandlerFunc(srv.handleCreateVouchers)))
	mux.Handle("POST /edit", strict(http.HandlerFunc(srv.handleEditVoucher)))

	// API (2 req/s, burst 10)
	mux.Handle("POST /status", api(http.HandlerFunc(srv.handleVoucherStatusBatch)))
	mux.Handle("GET /config", api(http.HandlerFunc(srv.handleConfig)))

	// Admin (strict rate limit + bearer auth)
	admin := func(h http.HandlerFunc) http.Handler {
		return strict(srv.adminAuth(h))
	}
	mux.Handle("GET /admin/ledger", admin(srv.handleLedger))
	mux.Handle("POST /admin/deposit", admin(srv.handleAdminDeposit))
	mux.Handle("POST /admin/withdraw", admin(srv.handleAdminWithdraw))

	// LNURL (5 req/s, burst 20)
	mux.Handle("POST /transfer", lnurl(http.HandlerFunc(srv.handleTransfer)))

	mux.Handle("GET /f/{pubKey}", lnurl(http.HandlerFunc(srv.handleLNURLPayVoucher)))
	mux.Handle("GET /w/{secret}", lnurl(http.HandlerFunc(srv.handleLNURLWithdraw)))
	mux.Handle("GET /fund/{pubKey}/callback", lnurl(http.HandlerFunc(srv.handleLNURLPayCallbackVoucher)))
	mux.Handle("GET /redeem/{secret}/callback", lnurl(http.HandlerFunc(srv.handleLNURLWithdrawCallback)))
	mux.Handle("GET /verify/{key}", lnurl(http.HandlerFunc(srv.handleLNURLVerify)))

	if err := http.ListenAndServe(":"+srv.cfg.port, corsMiddleware(mux)); err != nil {
		slog.Error("server", "err", err)
	}
}

// adminAuth gates /admin/* routes behind a bearer token. When ADMIN_SECRET is not
// configured the routes return 503 so a misconfigured deploy fails closed.
func (srv *Server) adminAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if srv.cfg.adminSecret == "" {
			lnurlError(w, http.StatusServiceUnavailable, "admin api is not configured")
			return
		}
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || subtle.ConstantTimeCompare([]byte(token), []byte(srv.cfg.adminSecret)) != 1 {
			lnurlError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next(w, r)
	}
}
