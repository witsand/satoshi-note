package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	spark "github.com/breez/breez-sdk-spark-go/breez_sdk_spark"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))

	srv, err := loadConfig()
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}
	srv.paymentSema = newPaymentSemaphore()
	srv.refundWake = make(chan struct{}, 1)
	slog.Info("config loaded")

	srv.ln, err = NewBreezClient(srv.cfg.mnemonic, srv.cfg.apiKey, srv.cfg.storageDirectory, srv.cfg.network, srv.cfg.maxConcurrentClaims)
	if err != nil {
		slog.Error("create breez client", "err", err)
		os.Exit(1)
	}
	srv.ln.AddEventListener(&SparkListener{srv: srv})
	slog.Info("breez client started")

	// Sync once up front so the SDK's local cache (balance, payment history) is
	// current before reconciling pending txs and before /ledger reads it. GetInfo
	// without EnsureSynced only ever reads that cache.
	_, rawSyncErr := srv.ln.SyncWallet(spark.SyncWalletRequest{})
	if err := sdkErr(rawSyncErr); err != nil {
		slog.Error("initial wallet sync", "err", err)
		os.Exit(1)
	}
	slog.Info("initial wallet sync complete")

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

	srv.checkPendingOperatorDeposits()
	srv.resolvePendingOperatorWithdraws()

	srv.resolvePendingRedeemTxs()

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
	// Stops the SDK's background tasks and unregisters listeners.
	if err := sdkErr(srv.ln.Disconnect()); err != nil {
		slog.Error("sdk disconnect", "err", err)
	}
}
