package main

import (
	"fmt"
	"log/slog"
	"time"

	spark "github.com/breez/breez-sdk-spark-go/breez_sdk_spark"
)

const fundTxLookbackSecs = 600

// resolvePendingRedeemTxs inspects redeem_txs left in TxPending state from a previous
// run. A pending row means the server crashed after deducting the voucher balance but
// before (or during) SendPayment — the payment outcome is unknown.
//
// Rows with a stored SDK payment id are resolved via GetPayment: a completed payment
// is marked confirmed; a failed payment is marked failed and the deducted balance is
// restored. Rows without a payment id (crash before SendPayment returned) are NOT
// resolved automatically — the balance stays deducted pending manual review. A manual
// retry of such a redeem is safe: SendPayment is called with a per-redeem idempotency
// key, so the SDK returns the original payment instead of paying twice.
func (srv *Server) resolvePendingRedeemTxs() {
	rows, err := srv.db.Query(
		`SELECT id, voucher_id, msat, ln_fee, payment_id, created_at FROM redeem_txs WHERE status = ?`, TxPending,
	)
	if err != nil {
		slog.Error("startup: check pending redeem txs", "err", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id, voucherID, msat, lnFee, createdAt int64
		var paymentID string
		if err := rows.Scan(&id, &voucherID, &msat, &lnFee, &paymentID, &createdAt); err != nil {
			slog.Error("startup: scan pending redeem tx", "err", err)
			return
		}

		if paymentID == "" {
			slog.Warn("startup: found pending redeem tx from previous run — payment outcome unknown, manual review required",
				"id", id,
				"voucher_id", voucherID,
				"amount_msat", msat,
				"created_at", createdAt,
			)
			continue
		}

		resp, rawErr := srv.ln.GetPayment(spark.GetPaymentRequest{PaymentId: paymentID})
		if err := sdkErr(rawErr); err != nil {
			slog.Warn("startup: could not resolve pending redeem tx via GetPayment — manual review required",
				"id", id, "payment_id", paymentID, "err", err)
			continue
		}

		switch resp.Payment.Status {
		case spark.PaymentStatusCompleted:
			var actualFeeMsat int64
			if resp.Payment.Fees != nil {
				actualFeeMsat = resp.Payment.Fees.Int64() * 1000
			}
			if err := srv.updateRedeemTx(id, TxConfirmed, lnFee-actualFeeMsat, actualFeeMsat, ""); err != nil {
				slog.Error("startup: mark resolved redeem tx confirmed", "id", id, "err", err)
				continue
			}
			slog.Info("startup: resolved pending redeem tx as confirmed",
				"id", id, "payment_id", paymentID, "amount_msat", msat)
		case spark.PaymentStatusFailed:
			if err := srv.updateRedeemTx(id, TxFailed, 0, 0, "payment failed (resolved at startup)"); err != nil {
				slog.Error("startup: mark resolved redeem tx failed", "id", id, "err", err)
				continue
			}
			// Restore the balance deducted before the payment attempt.
			if err := srv.addVoucherBalance(voucherID, msat+lnFee); err != nil {
				slog.Error("startup: restore voucher balance for failed redeem", "id", id, "voucher_id", voucherID, "err", err)
				continue
			}
			slog.Info("startup: resolved pending redeem tx as failed, balance restored",
				"id", id, "payment_id", paymentID, "voucher_id", voucherID, "amount_msat", msat)
		default:
			slog.Warn("startup: pending redeem tx still in flight at the SDK — manual review required",
				"id", id, "payment_id", paymentID, "status", resp.Payment.Status)
		}
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
