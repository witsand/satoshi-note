package main

import (
	"log/slog"
	"net/http"
	"os"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))

	cfg, err := loadConfig()
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}

	db, err := openDB(cfg.storageDirectory + "/satoshi_note.db")
	if err != nil {
		slog.Error("open database", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	srv := &Server{db: db, cfg: cfg}
	http.HandleFunc("POST /voucher/create", srv.handleCreateVoucher)
	http.HandleFunc("POST /voucher/create/{amount}", srv.handleCreateVouchers)

	slog.Info("listening", "port", cfg.port)
	if err := http.ListenAndServe(":"+cfg.port, nil); err != nil {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}
