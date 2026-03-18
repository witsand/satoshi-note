package main

import (
	"database/sql"

	spark "github.com/breez/breez-sdk-spark-go/breez_sdk_spark"
)

type TxStatus string

type Server struct {
	ln  *spark.BreezSdk
	db  *sql.DB
	cfg *Config
}

const (
	TxPending   TxStatus = "pending"
	TxConfirmed TxStatus = "confirmed"
	TxFailed    TxStatus = "failed"
)

type Voucher struct {
	ID                 int64  `json:"-"`
	Secret             string `json:"secret,omitempty"`
	ClaimLNURL         string `json:"claim_lnurl,omitempty"`
	PubKey             string `json:"pubkey,omitempty"`
	FundURLPrefix      string `json:"fund_url_prefix,omitempty"`
	BatchName          string `json:"batch_name,omitempty"`
	BatchID            string `json:"batch_id,omitempty"`
	BatchFundURLPrefix string `json:"batch_fund_url_prefix,omitempty"`
	WithdrawURLPrefix  string `json:"withdraw_url_prefix,omitempty"`
	RefundCode         string `json:"refund_code,omitempty"`
	RefundAfterSeconds int64  `json:"refund_after_seconds"`
	BalanceMsat        int64  `json:"balance_msat,omitempty"`
	Active             bool   `json:"active,omitempty"`
	SingleUse          bool   `json:"single_use,omitempty"`
	Refunded           bool   `json:"refunded,omitempty"`
	UpdatedAt          int64  `json:"updated_at,omitempty"`
}

type RefundTx struct {
	ID         int64
	RefundCode string
	AmountMsat int64
	DbTxFee    int64
	ActualFee  int64
	Refunded   bool
	ErrorMsg   string
	CreatedAt  int64
}

type AuditStats struct {
	TotalVouchers       int64 `json:"total_vouchers"`
	ActiveUnfunded      int64 `json:"active_unfunded"`
	ActiveFunded        int64 `json:"active_funded"`
	TotalRedeemed       int64 `json:"total_redeemed"`
	TotalRefunded       int64 `json:"total_refunded"`
	FailedRefunds       int64 `json:"failed_refunds"`
	ClaimableMsat       int64 `json:"claimable_msat"`
	RedeemedMsat        int64 `json:"redeemed_msat"`
	RefundedMsat        int64 `json:"refunded_msat"`
	PendingRefundMsat   int64 `json:"pending_refund_msat"`
	BreezBalanceMsat    int64 `json:"breez_balance_msat"`     // -1 if unavailable
	TotalDepositedMsat  int64 `json:"total_deposited_msat"`   // sum of confirmed fund_txs
	ExpiredNoRefundMsat int64 `json:"expired_no_refund_msat"` // absorbed sats: expired vouchers with no refund address
	SurplusMsat         int64 `json:"surplus_msat"`           // node balance minus all liabilities; -1 if Breez unavailable
	TotalDonations      int64 `json:"total_donations"`
	ConfirmedDonations  int64 `json:"confirmed_donations"`
	DonatedMsat         int64 `json:"donated_msat"`
}

type Donation struct {
	Key             string
	AmountMsat      int64
	FeeMsat         int64
	PR              string
	PaymentHash     string
	PaymentPreimage string
	Comment         string
	Status          TxStatus
	CreatedAt       int64
}

type FundTx struct {
	Key             string
	BatchID         string
	PubKey          string
	Msat            int64
	FeeMsat         int64
	PR              string
	PaymentHash     string
	PaymentPreimage string
	Status          TxStatus
	CreatedAt       int64
}
