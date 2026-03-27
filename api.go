package main

import (
	"log/slog"
	"net/http"
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
	strict := newRateLimiter(rate.Every(5*time.Second), 5).Middleware
	api := newRateLimiter(rate.Every(time.Second), 5).Middleware
	lnurl := newRateLimiter(rate.Every(time.Second), 5).Middleware

	mux := http.NewServeMux()

	// Static pages — no rate limit
	mux.Handle("/", http.FileServer(http.Dir("./static")))
	mux.HandleFunc("GET /{$}", srv.handleIndexPage)
	mux.HandleFunc("GET /redeem", srv.handleRedeemPage)
	mux.HandleFunc("GET /admin", srv.handleAdminPage)
	mux.HandleFunc("GET /manifest.json", srv.handleManifest)

	// Strict (1 req/2s, burst 5)
	mux.Handle("POST /voucher/create", strict(http.HandlerFunc(srv.handleCreateVouchers)))
	mux.Handle("POST /transfer", strict(http.HandlerFunc(srv.handleTransfer)))

	// API (2 req/s, burst 10)
	mux.Handle("GET /voucher/status/{pubKey}", api(http.HandlerFunc(srv.handleVoucherStatus)))
	mux.Handle("POST /voucher/status/batch", api(http.HandlerFunc(srv.handleVoucherStatusBatch)))
	mux.Handle("POST /leaderboard", api(http.HandlerFunc(srv.handleLeaderboard)))
	mux.Handle("GET /admin/stats", api(http.HandlerFunc(srv.handleAdminStats)))
	mux.Handle("GET /admin/recent", api(http.HandlerFunc(srv.handleAdminRecent)))
	mux.Handle("GET /config", api(http.HandlerFunc(srv.handleConfig)))

	// LNURL (5 req/s, burst 20)
	mux.Handle("GET /f/{pubKey}", lnurl(http.HandlerFunc(srv.handleLNURLPayVoucher)))
	mux.Handle("GET /w/{secret}", lnurl(http.HandlerFunc(srv.handleLNURLWithdraw)))
	mux.Handle("GET /fund/{pubKey}/callback", lnurl(http.HandlerFunc(srv.handleLNURLPayCallbackVoucher)))
	mux.Handle("GET /redeem/{secret}/callback", lnurl(http.HandlerFunc(srv.handleLNURLWithdrawCallback)))
	mux.Handle("GET /verify/{key}", lnurl(http.HandlerFunc(srv.handleLNURLVerify)))

	if err := http.ListenAndServe(":"+srv.cfg.port, corsMiddleware(mux)); err != nil {
		slog.Error("server", "err", err)
	}
}
