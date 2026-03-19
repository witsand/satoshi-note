package main

import (
	"log/slog"
	"net/http"
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
	mux := http.NewServeMux()

	// Static
	mux.Handle("/", http.FileServer(http.Dir("./static")))
	mux.HandleFunc("GET /{$}", srv.handleIndexPage)
	mux.HandleFunc("GET /redeem", srv.handleRedeemPage)
	mux.HandleFunc("GET /admin", srv.handleAdminPage)
	mux.HandleFunc("GET /manifest.json", srv.handleManifest)

	// App Api
	mux.HandleFunc("POST /voucher/create", srv.handleCreateVouchers)
	mux.HandleFunc("GET /voucher/status/{pubKey}", srv.handleVoucherStatus)
	mux.HandleFunc("GET /admin/stats", srv.handleAdminStats)
	mux.HandleFunc("GET /config", srv.handleConfig)

	// LNURL Step 1
	mux.HandleFunc("GET /donate", srv.handleDonate)
	mux.HandleFunc("GET /f/{pubKey}", srv.handleLNURLPayVoucher)
	mux.HandleFunc("GET /fb/{batchID}", srv.handleLNURLPayBatch)
	mux.HandleFunc("GET /w/{secret}", srv.handleLNURLWithdraw)

	// LNURL Step 2
	mux.HandleFunc("GET /donate/callback", srv.handleDonateCallback)
	mux.HandleFunc("GET /fund/{pubKey}/callback", srv.handleLNURLPayCallbackVoucher)
	mux.HandleFunc("GET /fund/batch/{batchID}/callback", srv.handleLNURLPayCallbackBatch)
	mux.HandleFunc("GET /redeem/{secret}/callback", srv.handleLNURLWithdrawCallback)

	// LUD-21 Verify
	mux.HandleFunc("GET /donate/verify/{key}", srv.handleDonateVerify)
	mux.HandleFunc("GET /verify/{key}", srv.handleLNURLVerify)

	if err := http.ListenAndServe(":"+srv.cfg.port, corsMiddleware(mux)); err != nil {
		slog.Error("server", "err", err)
	}
}
