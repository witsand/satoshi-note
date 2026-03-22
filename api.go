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
	// --- Strict sub-mux: admin write endpoints (1 req/2s, burst 5) ---
	strictMux := http.NewServeMux()
	strictMux.HandleFunc("POST /voucher/create", srv.handleCreateVouchers)
	strictHandler := newRateLimiter(rate.Every(2*time.Second), 5).Middleware(strictMux)

	// --- API sub-mux: read endpoints (2 req/s, burst 10) ---
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("GET /voucher/status/{pubKey}", srv.handleVoucherStatus)
	apiMux.HandleFunc("GET /admin/stats", srv.handleAdminStats)
	apiMux.HandleFunc("GET /admin/recent", srv.handleAdminRecent)
	apiMux.HandleFunc("GET /config", srv.handleConfig)
	apiHandler := newRateLimiter(rate.Every(500*time.Millisecond), 10).Middleware(apiMux)

	// --- LNURL sub-mux: wallet-facing endpoints (5 req/s, burst 20) ---
	lnurlMux := http.NewServeMux()
	// Step 1
	lnurlMux.HandleFunc("GET /donate", srv.handleDonate)
	lnurlMux.HandleFunc("GET /f/{pubKey}", srv.handleLNURLPayVoucher)
	lnurlMux.HandleFunc("GET /fb/{batchID}", srv.handleLNURLPayBatch)
	lnurlMux.HandleFunc("GET /w/{secret}", srv.handleLNURLWithdraw)
	// Step 2
	lnurlMux.HandleFunc("GET /donate/callback", srv.handleDonateCallback)
	lnurlMux.HandleFunc("GET /fund/{pubKey}/callback", srv.handleLNURLPayCallbackVoucher)
	lnurlMux.HandleFunc("GET /fund/batch/{batchID}/callback", srv.handleLNURLPayCallbackBatch)
	lnurlMux.HandleFunc("GET /redeem/{secret}/callback", srv.handleLNURLWithdrawCallback)
	// LUD-21 Verify
	lnurlMux.HandleFunc("GET /donate/verify/{key}", srv.handleDonateVerify)
	lnurlMux.HandleFunc("GET /verify/{key}", srv.handleLNURLVerify)
	lnurlHandler := newRateLimiter(rate.Every(200*time.Millisecond), 20).Middleware(lnurlMux)

	// --- Main mux ---
	mux := http.NewServeMux()

	// Static pages — no rate limit
	// Note: GET /redeem and GET /admin are exact matches; /redeem/ and /admin/ prefixes
	// route to their respective rate-limited sub-muxes without ambiguity.
	mux.Handle("/", http.FileServer(http.Dir("./static")))
	mux.HandleFunc("GET /{$}", srv.handleIndexPage)
	mux.HandleFunc("GET /redeem", srv.handleRedeemPage)
	mux.HandleFunc("GET /admin", srv.handleAdminPage)
	mux.HandleFunc("GET /manifest.json", srv.handleManifest)

	// Strict
	mux.Handle("POST /voucher/create", strictHandler)

	// API
	mux.Handle("/voucher/status/", apiHandler)
	mux.Handle("/admin/", apiHandler)
	mux.Handle("/config", apiHandler)

	// LNURL — each path prefix forwards to the same rate-limited lnurlMux
	mux.Handle("GET /donate", lnurlHandler)
	mux.Handle("/donate/", lnurlHandler)
	mux.Handle("/f/", lnurlHandler)
	mux.Handle("/fb/", lnurlHandler)
	mux.Handle("/fund/", lnurlHandler)
	mux.Handle("/w/", lnurlHandler)
	mux.Handle("/redeem/", lnurlHandler)
	mux.Handle("/verify/", lnurlHandler)

	if err := http.ListenAndServe(":"+srv.cfg.port, corsMiddleware(mux)); err != nil {
		slog.Error("server", "err", err)
	}
}
