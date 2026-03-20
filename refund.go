package main

import (
	"fmt"
	"log/slog"
	"time"

	spark "github.com/breez/breez-sdk-spark-go/breez_sdk_spark"
)

func (srv *Server) runRefundWorker() {
	srv.processRefunds() // run once on startup to catch missed refunds
	ticker := time.NewTicker(time.Duration(srv.cfg.refundWorkerIntervalSeconds) * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		srv.processRefunds()
	}
}

func (srv *Server) processRefunds() {
	slog.Info("refund worker: checking for expired vouchers")

	vouchers, err := srv.getExpiredVouchersWithBalance()
	if err != nil {
		slog.Error("refund worker: get expired vouchers", "err", err)
		return
	}

	if len(vouchers) == 0 {
		slog.Info("refund worker: no expired vouchers found")
		return
	}

	// Group by refund_code.
	type group struct {
		ids        []int64
		totalMsat  int64
		refundCode string
	}
	groups := make(map[string]*group)
	for _, v := range vouchers {
		g, ok := groups[v.RefundCode]
		if !ok {
			g = &group{refundCode: v.RefundCode}
			groups[v.RefundCode] = g
		}
		g.ids = append(g.ids, v.ID)
		g.totalMsat += v.BalanceMsat
	}

	cfg := srv.cfg

	for refundCode, g := range groups {
		dbTxFee := g.totalMsat * cfg.redeemFeeBPS / 10000 / 1000 * 1000 + 1000
		if dbTxFee < cfg.minRedeemFeeMsat {
			dbTxFee = cfg.minRedeemFeeMsat / 1000 * 1000
		}
		netMsat := g.totalMsat - dbTxFee

		dbTx, err := srv.db.Begin()
		if err != nil {
			slog.Error("refund worker: begin tx", "refund_code", refundCode, "err", err)
			continue
		}

		var refunded bool
		var errorMsg string
		if netMsat < cfg.minRedeemAmountMsat {
			refunded = true
			errorMsg = "below minimum"
		}

		refundTxID, err := srv.insertRefundTx(dbTx, refundCode, netMsat, dbTxFee, refunded, errorMsg)
		if err != nil {
			slog.Error("refund worker: insert refund tx", "refund_code", refundCode, "err", err)
			dbTx.Rollback()
			continue
		}

		if err := srv.markVouchersRefunded(dbTx, g.ids, refundTxID); err != nil {
			slog.Error("refund worker: mark vouchers refunded", "refund_code", refundCode, "err", err)
			dbTx.Rollback()
			continue
		}

		if err := dbTx.Commit(); err != nil {
			slog.Error("refund worker: commit", "refund_code", refundCode, "err", err)
			continue
		}

		if refunded {
			slog.Info("refund worker: dust voucher skipped", "refund_code", refundCode, "net_msat", netMsat)
		}
	}

	// Attempt payment for all pending refund txs.
	pending, err := srv.getPendingRefundTxs()
	if err != nil {
		slog.Error("refund worker: get pending refund txs", "err", err)
		return
	}

	for _, rt := range pending {
		if err := srv.payRefund(rt); err != nil {
			slog.Error("refund worker: pay refund failed", "id", rt.ID, "refund_code", rt.RefundCode, "err", err)
		} else {
			slog.Info("refund worker: refund paid", "id", rt.ID, "refund_code", rt.RefundCode, "amount_msat", rt.AmountMsat)
		}
	}
}

func (srv *Server) payRefund(rt RefundTx) error {
	inputType, rawErr := srv.ln.Parse(rt.RefundCode)
	if err := sdkErr(rawErr); err != nil {
		return fmt.Errorf("parse refund_code: %w", err)
	}

	var payRequest spark.LnurlPayRequestDetails
	switch v := inputType.(type) {
	case spark.InputTypeLightningAddress:
		payRequest = v.Field0.PayRequest
	case spark.InputTypeLnurlPay:
		payRequest = v.Field0
	default:
		return fmt.Errorf("unsupported refund_code type: %T", inputType)
	}

	amountSats := uint64(rt.AmountMsat / 1000)

	prepResp, rawPrepErr := srv.ln.PrepareLnurlPay(spark.PrepareLnurlPayRequest{
		AmountSats: amountSats,
		PayRequest: payRequest,
	})
	if err := sdkErr(rawPrepErr); err != nil {
		markErr := srv.markRefundTxFailed(rt.ID, err.Error())
		if markErr != nil {
			slog.Error("refund worker: mark refund tx failed", "id", rt.ID, "err", markErr)
		}
		return fmt.Errorf("prepare lnurl pay: %w", err)
	}

	estimateFeeMsat := int64(prepResp.FeeSats) * 1000
	if estimateFeeMsat > rt.DbTxFee {
		markErr := srv.markRefundTxFailed(rt.ID, "routing fee too high")
		if markErr != nil {
			slog.Error("refund worker: mark refund tx failed", "id", rt.ID, "err", markErr)
		}
		return fmt.Errorf("routing fee %d msat exceeds db_tx_fee %d msat", estimateFeeMsat, rt.DbTxFee)
	}

	lnurlPayResp, rawPayErr := srv.ln.LnurlPay(spark.LnurlPayRequest{
		PrepareResponse: prepResp,
	})
	if err := sdkErr(rawPayErr); err != nil {
		markErr := srv.markRefundTxFailed(rt.ID, err.Error())
		if markErr != nil {
			slog.Error("refund worker: mark refund tx failed", "id", rt.ID, "err", markErr)
		}
		return fmt.Errorf("lnurl pay: %w", err)
	}

	var actualFeeMsat int64
	if lnurlPayResp.Payment.Fees != nil {
		actualFeeMsat = lnurlPayResp.Payment.Fees.Int64() * 1000
	}
	if err := srv.markRefundTxPaid(rt.ID, rt.DbTxFee-actualFeeMsat, actualFeeMsat); err != nil {
		slog.Error("refund worker: mark refund tx paid", "id", rt.ID, "err", err)
	}

	return nil
}
