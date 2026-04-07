package main

import (
	"fmt"
	"log/slog"
	"time"

	spark "github.com/breez/breez-sdk-spark-go/breez_sdk_spark"
)

func (srv *Server) runRefundWorker() {
	// Run once on startup to catch any missed refunds.
	srv.processExpiredRefunds()
	srv.processRegularRefunds()

	const maxSleep = time.Hour
	for {
		nextAt, err := srv.nextRefundAt()
		var wait time.Duration
		if err != nil || nextAt == nil {
			wait = maxSleep
		} else {
			wait = time.Until(*nextAt)
			if wait <= 0 {
				srv.processExpiredRefunds()
				srv.processRegularRefunds()
				continue
			}
			if wait > maxSleep {
				wait = maxSleep
			}
		}

		select {
		case <-time.After(wait):
		case <-srv.refundWake:
		}

		srv.processExpiredRefunds()
		srv.processRegularRefunds()
	}
}

// splitBalance distributes balanceMsat across codes proportional to their shares.
// Any remainder (dust from integer division) is added to the last code.
func splitBalance(balanceMsat int64, codes []VoucherRefundCode) map[string]int64 {
	var totalShares int64
	for _, c := range codes {
		totalShares += c.Share
	}
	out := make(map[string]int64, len(codes))
	var allocated int64
	for i, c := range codes {
		var alloc int64
		if i == len(codes)-1 {
			alloc = balanceMsat - allocated
		} else {
			alloc = balanceMsat * c.Share / totalShares
		}
		out[c.RefundCode] += alloc
		allocated += alloc
	}
	return out
}

// nextRegularRefundAnchor returns the last past-due scheduled anchor time,
// advancing by intervalSecs until the next occurrence would be in the future.
// This pins the refund schedule to firstAt regardless of processing delays.
func nextRegularRefundAnchor(firstAt, lastAt, intervalSecs, now int64) int64 {
	anchor := firstAt
	if lastAt > 0 {
		anchor = lastAt
	}
	for anchor <= now {
		anchor += intervalSecs
	}
	return anchor - intervalSecs
}

func (srv *Server) processExpiredRefunds() {
	slog.Info("refund worker: checking for expired vouchers")
	defer srv.doPendingRefunds()

	vouchers, err := srv.getExpiredVouchersWithBalance()
	if err != nil {
		slog.Error("refund worker: get expired vouchers", "err", err)
		return
	}

	if len(vouchers) == 0 {
		slog.Info("refund worker: no expired vouchers found")
		return
	}

	// Batch-fetch refund code splits for all expired vouchers.
	ids := make([]int64, len(vouchers))
	for i, v := range vouchers {
		ids[i] = v.ID
	}
	refundCodesMap, err := srv.getRefundCodesForVouchers(ids)
	if err != nil {
		slog.Error("refund worker: get refund codes", "err", err)
		return
	}

	// Group by refund_code, splitting each voucher's balance proportionally.
	type group struct {
		ids        []int64
		totalMsat  int64
		refundCode string
	}
	groups := make(map[string]*group)
	for _, v := range vouchers {
		codes := refundCodesMap[v.ID]
		splits := splitBalance(v.BalanceMsat, codes)
		for refundCode, amount := range splits {
			g, ok := groups[refundCode]
			if !ok {
				g = &group{refundCode: refundCode}
				groups[refundCode] = g
			}
			g.ids = append(g.ids, v.ID)
			g.totalMsat += amount
		}
	}

	cfg := srv.cfg

	for refundCode, g := range groups {
		dbTxFee := srv.calculateRedeemFee(g.totalMsat)
		netMsat := g.totalMsat - dbTxFee

		if netMsat < cfg.minRedeemAmountMsat {
			continue
		}

		dbTx, err := srv.db.Begin()
		if err != nil {
			slog.Error("refund worker: begin tx", "refund_code", refundCode, "err", err)
			continue
		}

		refundTxID, err := srv.insertRefundTx(dbTx, refundCode, netMsat, dbTxFee)
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
	}
}

func (srv *Server) processRegularRefunds() {
	slog.Info("refund worker: checking for regular refunds")
	defer srv.doPendingRefunds()

	now := time.Now().Unix()

	vouchers, err := srv.getRegularRefundDueVouchers()
	if err != nil {
		slog.Error("refund worker: get regular refund vouchers", "err", err)
		return
	}

	if len(vouchers) == 0 {
		slog.Info("refund worker: no regular refunds due")
		return
	}

	// Compute the drift-free anchor for each voucher and partition by balance.
	type regularGroup struct {
		entries    []regularRefundEntry
		totalMsat  int64
		refundCode string
	}
	groups := make(map[string]*regularGroup)
	var noBalanceEntries []regularRefundEntry

	ids := make([]int64, len(vouchers))
	for i, v := range vouchers {
		ids[i] = v.ID
	}
	refundCodesMap, err := srv.getRefundCodesForVouchers(ids)
	if err != nil {
		slog.Error("refund worker: get refund codes for regular", "err", err)
		return
	}

	for _, v := range vouchers {
		newLastAt := nextRegularRefundAnchor(v.FirstAt, v.LastAt, v.IntervalSecs, now)
		entry := regularRefundEntry{ID: v.ID, NewLastAt: newLastAt}

		if v.BalanceMsat == 0 {
			noBalanceEntries = append(noBalanceEntries, entry)
			continue
		}

		codes := refundCodesMap[v.ID]
		splits := splitBalance(v.BalanceMsat, codes)
		for refundCode, amount := range splits {
			g, ok := groups[refundCode]
			if !ok {
				g = &regularGroup{refundCode: refundCode}
				groups[refundCode] = g
			}
			g.entries = append(g.entries, entry)
			g.totalMsat += amount
		}
	}

	// Advance schedule for zero-balance vouchers without creating a payment.
	if len(noBalanceEntries) > 0 {
		if err := advanceRegularRefundTime(srv.db, noBalanceEntries); err != nil {
			slog.Error("refund worker: advance regular refund time", "err", err)
		}
	}

	cfg := srv.cfg

	for refundCode, g := range groups {
		dbTxFee := srv.calculateRedeemFee(g.totalMsat)
		netMsat := g.totalMsat - dbTxFee

		if netMsat < cfg.minRedeemAmountMsat {
			// Still advance the schedule even if amount is too small to send.
			if err := advanceRegularRefundTime(srv.db, g.entries); err != nil {
				slog.Error("refund worker: advance regular refund time (below min)", "err", err)
			}
			continue
		}

		dbTx, err := srv.db.Begin()
		if err != nil {
			slog.Error("refund worker: begin regular tx", "refund_code", refundCode, "err", err)
			continue
		}

		if _, err := srv.insertRefundTx(dbTx, refundCode, netMsat, dbTxFee); err != nil {
			slog.Error("refund worker: insert regular refund tx", "refund_code", refundCode, "err", err)
			dbTx.Rollback()
			continue
		}

		if err := markVouchersRegularRefunded(dbTx, g.entries); err != nil {
			slog.Error("refund worker: mark vouchers regular refunded", "refund_code", refundCode, "err", err)
			dbTx.Rollback()
			continue
		}

		if err := dbTx.Commit(); err != nil {
			slog.Error("refund worker: commit regular", "refund_code", refundCode, "err", err)
			continue
		}
	}
}

func (srv *Server) doPendingRefunds() {
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
	srv.paymentSema.acquireForRefund()
	defer srv.paymentSema.releaseAfter(srv.cfg.paymentCooldown)

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
	comment := "Voucher Refunds"

	var commentPtr *string
	if payRequest.CommentAllowed > 0 {
		if len(comment) > int(payRequest.CommentAllowed) {
			truncated := comment[:payRequest.CommentAllowed]
			commentPtr = &truncated
		} else {
			commentPtr = &comment
		}
	}

	prepResp, rawPrepErr := srv.ln.PrepareLnurlPay(spark.PrepareLnurlPayRequest{
		AmountSats: amountSats,
		PayRequest: payRequest,
		Comment:    commentPtr,
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
