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
	mux.Handle("/", http.FileServer(http.Dir("./static")))
	mux.HandleFunc("POST /voucher/create", srv.handleCreateVouchers)
	mux.HandleFunc("GET /f/{pubKey}", srv.handleLNURLPayVoucher)
	mux.HandleFunc("GET /fb/{batchID}", srv.handleLNURLPayBatch)
	mux.HandleFunc("GET /fund/{pubKey}/callback", srv.handleLNURLPayCallbackVoucher)
	mux.HandleFunc("GET /fund/batch/{batchID}/callback", srv.handleLNURLPayCallbackBatch)
	mux.HandleFunc("GET /fv/{key}", srv.handleLNURLVerify)
	mux.HandleFunc("GET /w/{secret}", srv.handleLNURLWithdraw)
	mux.HandleFunc("GET /withdraw/{secret}/callback", srv.handleLNURLWithdrawCallback)

	if err := http.ListenAndServe(":"+srv.cfg.port, corsMiddleware(mux)); err != nil {
		slog.Error("server", "err", err)
	}
}
