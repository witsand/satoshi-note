package main

import (
	"fmt"
	"log/slog"
	"time"

	spark "github.com/breez/breez-sdk-spark-go/breez_sdk_spark"
)

const fundTxLookbackSecs = 600

// warnPendingRedeemTxs logs a warning for any redeem_txs left in TxPending state
// from a previous run. A pending row means the server crashed after deducting the
// voucher balance but before (or during) SendPayment — the payment outcome is unknown.
// These rows are NOT automatically resolved: the voucher balance remains deducted
// (server holds the funds) until an operator investigates and manually restores if needed.
func (srv *Server) warnPendingRedeemTxs() {
	rows, err := srv.db.Query(
		`SELECT id, voucher_id, msat, created_at FROM redeem_txs WHERE status = ?`, TxPending,
	)
	if err != nil {
		slog.Error("startup: check pending redeem txs", "err", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id, voucherID, msat, createdAt int64
		if err := rows.Scan(&id, &voucherID, &msat, &createdAt); err != nil {
			slog.Error("startup: scan pending redeem tx", "err", err)
			return
		}
		slog.Warn("startup: found pending redeem tx from previous run — payment outcome unknown, manual review required",
			"id", id,
			"voucher_id", voucherID,
			"amount_msat", msat,
			"created_at", createdAt,
		)
	}
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

	ps, err := srv.getPaymentsCompleted(since - fundTxLookbackSecs) // Get payments prior to earliest pending
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
					return fmt.Errorf("update fund tx confirmed (pr=%s): %w", tx.PR, err)
				}
			}
		}
	}

	return nil
}
