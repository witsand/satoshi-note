package main

import (
	"log/slog"
	"time"

	spark "github.com/breez/breez-sdk-spark-go/breez_sdk_spark"
)

func (srv *Server) checkPendingDonations() error {
	ds, err := srv.getPendingDonations()
	if err != nil {
		return err
	}

	if len(ds) == 0 {
		return nil
	}

	donMap := make(map[string]*Donation)
	since := uint64(time.Now().Unix())
	for i, d := range ds {
		donMap[d.PR] = &ds[i]
		if since > uint64(d.CreatedAt) {
			since = uint64(d.CreatedAt)
		}
	}

	ps, err := srv.getPaymentsCompleted(since - 600)
	if err != nil {
		return err
	}

	for _, p := range ps {
		if p.Details == nil {
			continue
		}
		if details, ok := (*p.Details).(spark.PaymentDetailsLightning); ok {
			if don, yes := donMap[details.Invoice]; yes {
				amountMsat := p.Amount.Int64() * 1000
				feeMsat := p.Fees.Int64() * 1000
				paymentHash := details.HtlcDetails.PaymentHash
				var preimage string
				if details.HtlcDetails.Preimage != nil {
					preimage = *details.HtlcDetails.Preimage
				}
				if err := srv.markDonationConfirmed(don.Key, paymentHash, preimage, amountMsat, feeMsat); err != nil {
					slog.Error("mark donation confirmed at startup", "err", err)
					continue
				}
			}
		}
	}

	return nil
}

func (srv *Server) checkPendingFundTXs() error {
	txs, err := srv.getPendingFundTxs()
	if err != nil {
		return err
	}

	if len(txs) == 0 {
		return nil
	}

	txMap := make(map[string]*FundTx)
	since := uint64(time.Now().Unix())
	for _, tx := range txs {
		txMap[tx.PR] = &tx

		if since > uint64(tx.CreatedAt) {
			since = uint64(tx.CreatedAt)
		}
	}

	ps, err := srv.getPaymentsCompleted(since - 600) // Get payments 10 minutes prior to last pengding payment
	if err != nil {
		return err
	}

	for _, p := range ps {
		if p.Details == nil {
			continue
		}

		if details, ok := (*p.Details).(spark.PaymentDetailsLightning); ok {
			if tx, yes := txMap[details.Invoice]; yes {
				tx.Msat = p.Amount.Int64() * 1000
				tx.FeeMsat = p.Fees.Int64() * 1000
				tx.PaymentHash = details.HtlcDetails.PaymentHash
				tx.PaymentPreimage = *details.HtlcDetails.Preimage

				if err := srv.updateFundTxConfirmed(tx); err != nil {
					slog.Error("updated fund tx confirmed failed", "err", err)
					continue
				}
			}
		}
	}

	return nil
}
