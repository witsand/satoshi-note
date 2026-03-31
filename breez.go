package main

import (
	"fmt"
	"log/slog"
	"math"
	"time"

	spark "github.com/breez/breez-sdk-spark-go/breez_sdk_spark"
)

type SparkListener struct {
	srv *Server
}

func (l *SparkListener) OnEvent(e spark.SdkEvent) {
	// slog.Info("sdk event", "type", fmt.Sprintf("%T", e))

	ev, ok := e.(spark.SdkEventPaymentSucceeded)
	if !ok {
		return
	}

	if ev.Payment.Details == nil {
		return
	}

	details, ok := (*ev.Payment.Details).(spark.PaymentDetailsLightning)
	if !ok {
		return
	}

	tx, err := l.srv.getFundTxByPR(details.Invoice)
	if err == nil {
		if ev.Payment.Amount != nil {
			tx.Msat = ev.Payment.Amount.Int64() * 1000
		}
		if ev.Payment.Fees != nil {
			tx.FeeMsat = ev.Payment.Fees.Int64() * 1000
		}
		tx.PaymentHash = details.HtlcDetails.PaymentHash
		if details.HtlcDetails.Preimage != nil {
			tx.PaymentPreimage = *details.HtlcDetails.Preimage
		}
		if err := l.srv.updateFundTxConfirmed(tx); err != nil {
			slog.Error("update fund tx confirmed", "err", err)
		}
	}
}

// sdkErr unwraps the typed nil that uniffiRustCallAsync produces when Rust
// returns no error: *SdkError(nil) satisfies the error interface but the
// pointer is nil, so a plain "!= nil" check incorrectly reports failure.
func sdkErr(err error) error {
	if se, ok := err.(*spark.SdkError); ok {
		return se.AsError()
	}
	return err
}

func NewBreezClient(mnemonic, apiKey, storageDirectory string, network spark.Network) (*spark.BreezSdk, error) {
	cfg := spark.DefaultConfig(network)
	cfg.ApiKey = &apiKey

	var seed spark.Seed = spark.SeedMnemonic{
		Mnemonic:   mnemonic,
		Passphrase: nil,
	}

	connectRequest := spark.ConnectRequest{
		Config:     cfg,
		Seed:       seed,
		StorageDir: storageDirectory,
	}

	s, err := spark.Connect(connectRequest)
	if s == nil {
		if err != nil {
			return nil, fmt.Errorf("SDK Connect: %w", err)
		}
		return nil, fmt.Errorf("SDK Connect: returned nil SDK with no error")
	}

	return s, nil
}

func Int64ToUint64(i int64) (uint64, error) {
	if i < 0 {
		return 0, fmt.Errorf("cannot convert negative int64 to uint64: %d", i)
	}
	return uint64(i), nil
}

func Int64ToUint32(i int64) (uint32, error) {
	if i < 0 || i > math.MaxUint32 {
		return 0, fmt.Errorf("value out of uint32 range: %d", i)
	}
	return uint32(i), nil
}

func (srv *Server) getCallbackBolt11(tx *FundTx, description string) error {
	tx.CreatedAt = time.Now().Unix()
	sat := tx.Msat / 1000
	tx.Msat = sat * 1000
	usat, err := Int64ToUint64(sat)
	if err != nil {
		return err
	}
	uexpiry, err := Int64ToUint32(srv.cfg.invoiceExpirySeconds)
	if err != nil {
		return err
	}
	resp, rawErr := srv.ln.ReceivePayment(spark.ReceivePaymentRequest{
		PaymentMethod: spark.ReceivePaymentMethodBolt11Invoice{
			AmountSats:  &usat,
			Description: description,
			ExpirySecs:  &uexpiry,
		},
	})
	if err := sdkErr(rawErr); err != nil {
		return fmt.Errorf("create invoice: %w", err)
	}

	if resp.Fee != nil {
		tx.FeeMsat = resp.Fee.Int64() * 1000
	}
	tx.PR = resp.PaymentRequest

	return nil
}

// getPaymentsCompleted searches completed receive payments since the given timestamp.
func (srv *Server) getPaymentsCompleted(since uint64) ([]spark.Payment, error) {
	typeFilter := []spark.PaymentType{spark.PaymentTypeReceive}
	statusFilter := []spark.PaymentStatus{spark.PaymentStatusCompleted}
	var assetFilter spark.AssetFilter = spark.AssetFilterBitcoin{}
	listResp, err := srv.ln.ListPayments(spark.ListPaymentsRequest{
		TypeFilter:    &typeFilter,
		StatusFilter:  &statusFilter,
		AssetFilter:   &assetFilter,
		FromTimestamp: &since,
	})

	return listResp.Payments, sdkErr(err)
}
