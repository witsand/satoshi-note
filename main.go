package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))

	srv, err := loadConfig()
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}
	slog.Info("config loaded")

	srv.ln, err = NewBreezClient(srv.cfg.mnemonic, srv.cfg.apiKey, srv.cfg.storageDirectory, srv.cfg.network)
	if err != nil {
		slog.Error("create breez client", "err", err)
		os.Exit(1)
	}
	srv.ln.AddEventListener(&SparkListener{srv: srv})
	slog.Info("breez client started")

	srv.db, err = openDB(srv.cfg.storageDirectory + "/satoshi_note.db")
	if err != nil {
		slog.Error("open database", "err", err)
		os.Exit(1)
	}
	defer srv.db.Close()
	slog.Info("database opened")

	err = srv.checkPendingFundTXs()
	if err != nil {
		slog.Error("check pending fund txs", "err", err)
		os.Exit(1)
	}
	slog.Info("pending funding payments caught up")

	if err = srv.checkPendingDonations(); err != nil {
		slog.Error("check pending donations", "err", err)
		os.Exit(1)
	}
	slog.Info("pending donations caught up")

	if srv.cfg.refundActive {
		go srv.runRefundWorker()
	}
	slog.Info("refund worker started")

	go srv.ServeAPI()
	slog.Info("listening", "port", srv.cfg.port)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	slog.Info("shutting down")
}
