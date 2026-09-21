package main

import (
	"fmt"
	"log/slog"
	"math"
	"math/big"
	"time"

	spark "github.com/breez/breez-sdk-spark-go/breez_sdk_spark"
)

type SparkListener struct {
	srv *Server
}

func (l *SparkListener) OnEvent(e spark.SdkEvent) {
	switch ev := e.(type) {
	case spark.SdkEventPaymentSucceeded:
		l.onPaymentSucceeded(ev.Payment)
	case spark.SdkEventSynced:
		// The SDK cache (what GetInfo reads) is now fresh from the network.
		slog.Info("sdk event: wallet synced")
	case spark.SdkEventPaymentPending:
		slog.Info("sdk event: payment in flight", "payment_id", ev.Payment.Id)
	case spark.SdkEventPaymentFailed:
		slog.Warn("sdk event: payment failed",
			"payment_id", ev.Payment.Id,
			"amount_sats", u128OrNil(ev.Payment.Amount),
			"details_type", paymentDetailsType(ev.Payment.Details),
		)
	case spark.SdkEventNewDeposits:
		slog.Info("sdk event: on-chain deposits detected", "count", len(ev.NewDeposits))
	case spark.SdkEventClaimedDeposits:
		slog.Info("sdk event: deposits claimed", "count", len(ev.ClaimedDeposits))
	case spark.SdkEventUnclaimedDeposits:
		for _, d := range ev.UnclaimedDeposits {
			slog.Warn("sdk event: deposit could not be claimed",
				"txid", d.Txid, "vout", d.Vout, "amount_sats", d.AmountSats,
				"claim_error", d.ClaimError)
		}
	case spark.SdkEventAutoOptimization:
		// Background leaf optimizer progress; locks leaves briefly (balance flicker).
		slog.Debug("sdk event: auto leaf optimization")
	default:
		slog.Debug("sdk event", "type", fmt.Sprintf("%T", e))
	}
}

// onPaymentSucceeded credits a voucher for a paid fund invoice, or confirms an
// operator deposit when the invoice is not a voucher fund invoice. The SDK refreshes
// its cached balance before emitting this event, so GetInfo reflects the payment.
func (l *SparkListener) onPaymentSucceeded(p spark.Payment) {
	if p.Details == nil {
		return
	}

	details, ok := (*p.Details).(spark.PaymentDetailsLightning)
	if !ok {
		return
	}

	tx, err := l.srv.getFundTxByPR(details.Invoice)
	if err == nil {
		if p.Amount != nil {
			// The SDK credits us net of the receive fee (the SSP deducts it before the
			// payment reaches our leaves), so Payment.Amount is what actually arrived.
			// The actual receive fee is the invoice amount minus what arrived. Keep
			// tx.Msat as the invoice amount so the row reflects what the funder paid;
			// updateFundBalance credits tx.Msat - tx.FeeMsat = net either way.
			if fee := tx.Msat - p.Amount.Int64()*1000; fee >= 0 {
				tx.FeeMsat = fee
			}
		}
		tx.PaymentHash = details.HtlcDetails.PaymentHash
		if details.HtlcDetails.Preimage != nil {
			tx.PaymentPreimage = *details.HtlcDetails.Preimage
		}
		if err := l.srv.updateFundTxConfirmed(tx); err != nil {
			slog.Error("update fund tx confirmed", "err", err)
		}
		return
	}

	// Not a voucher fund invoice — try an operator deposit for the same invoice.
	var amountMsat int64
	if p.Amount != nil {
		amountMsat = p.Amount.Int64() * 1000
	}
	preimage := ""
	if details.HtlcDetails.Preimage != nil {
		preimage = *details.HtlcDetails.Preimage
	}
	if err := l.srv.confirmOperatorDepositByPR(details.Invoice, amountMsat, details.HtlcDetails.PaymentHash, preimage); err == nil {
		slog.Info("operator deposit confirmed", "pr", details.Invoice, "amount_msat", amountMsat)
	}
}

// u128OrNil renders a u128 amount for logging, tolerating nil.
func u128OrNil(v *big.Int) any {
	if v == nil {
		return nil
	}
	return v.Int64()
}

// sendCostMsat returns the true msat cost of a completed send beyond the invoice
// amount: the wallet's total spend (Payment.Amount + Payment.Fees) minus the
// invoice amount. For a Lightning send the fee is reported in Payment.Fees and
// Payment.Amount is the invoice amount; for a Spark-transfer send Payment.Fees is
// 0 and the transfer fee is baked into Payment.Amount (the SDK's transfer→payment
// conversion only separates the fee for LightningSendRequest). Reading
// Payment.Fees alone would report 0 for Spark transfers and under-book the cost,
// leaking the fee from the ledger identity (explained > wallet by the unbooked
// fee, driving imbalance negative on every Spark-paid redeem/refund/withdraw).
func sendCostMsat(p spark.Payment, invoiceAmountMsat int64) int64 {
	if p.Amount == nil {
		// No amount recorded — fall back to the reported fee alone.
		if p.Fees != nil {
			return p.Fees.Int64() * 1000
		}
		return 0
	}
	totalMsat := p.Amount.Int64() * 1000
	if p.Fees != nil {
		totalMsat += p.Fees.Int64() * 1000
	}
	if cost := totalMsat - invoiceAmountMsat; cost > 0 {
		return cost
	}
	return 0
}

// paymentDetailsType names the payment details variant for logging.
func paymentDetailsType(details *spark.PaymentDetails) string {
	if details == nil {
		return "none"
	}
	return fmt.Sprintf("%T", *details)
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

func NewBreezClient(mnemonic, apiKey, storageDirectory string, network spark.Network, maxConcurrentClaims uint32) (*spark.BreezSdk, error) {
	cfg := spark.DefaultConfig(network)
	cfg.ApiKey = &apiKey
	// Default is 4; we are a server receiving many concurrent payments.
	cfg.MaxConcurrentClaims = maxConcurrentClaims

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
// Pages through the full history window: without an explicit limit the SDK default
// could truncate the result and a pending fund tx would never be caught up.
func (srv *Server) getPaymentsCompleted(since uint64) ([]spark.Payment, error) {
	typeFilter := []spark.PaymentType{spark.PaymentTypeReceive}
	statusFilter := []spark.PaymentStatus{spark.PaymentStatusCompleted}
	var assetFilter spark.AssetFilter = spark.AssetFilterBitcoin{}

	const pageSize = 500
	var all []spark.Payment
	for offset := uint32(0); ; offset += pageSize {
		limit := uint32(pageSize)
		listResp, rawErr := srv.ln.ListPayments(spark.ListPaymentsRequest{
			TypeFilter:    &typeFilter,
			StatusFilter:  &statusFilter,
			AssetFilter:   &assetFilter,
			FromTimestamp: &since,
			Offset:        &offset,
			Limit:         &limit,
		})
		if err := sdkErr(rawErr); err != nil {
			return nil, err
		}
		all = append(all, listResp.Payments...)
		if len(listResp.Payments) < pageSize {
			break
		}
	}

	return all, nil
}
