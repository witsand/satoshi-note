package main

import (
	"database/sql"

	spark "github.com/breez/breez-sdk-spark-go/breez_sdk_spark"
)

type TxStatus string

type Server struct {
	ln          *spark.BreezSdk
	db          *sql.DB
	cfg         *Config
	paymentSema *paymentSemaphore
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
	BatchID            string `json:"batch_id,omitempty"`
	WithdrawURLPrefix  string `json:"withdraw_url_prefix,omitempty"`
	RefundCode         string `json:"refund_code,omitempty"`
	RefundAfterSeconds int64  `json:"refund_after_seconds"`
	BalanceMsat        int64  `json:"balance_msat,omitempty"`
	Active             bool   `json:"active,omitempty"`
	SingleUse          bool   `json:"single_use,omitempty"`
	TransfersOnly      bool   `json:"transfers_only,omitempty"`
	MaxRedeemMsat      int64  `json:"max_redeem_msat,omitempty"`
	UniqueRedemptions  bool   `json:"unique_redemptions,omitempty"`
	Refunded           bool   `json:"refunded,omitempty"`
	UpdatedAt          int64  `json:"updated_at,omitempty"`
	AbsoluteExpiry     bool   `json:"absolute_expiry,omitempty"`
}

type RefundTx struct {
	ID         int64  `json:"id"`
	RefundCode string `json:"refund_code"`
	AmountMsat int64  `json:"amount_msat"`
	DbTxFee    int64  `json:"db_tx_fee"`
	ActualFee  int64  `json:"actual_fee"`
	Refunded   bool   `json:"refunded"`
	ErrorMsg   string `json:"error_msg"`
	CreatedAt  int64  `json:"created_at"`
}

type RedeemTx struct {
	ID          int64    `json:"id"`
	VoucherID   int64    `json:"voucher_id"`
	AmountMsat  int64    `json:"amount_msat"`
	LnFee       int64    `json:"ln_fee"`
	DbTxFee     int64    `json:"db_tx_fee"`
	ActualLnFee int64    `json:"actual_ln_fee"`
	Status      TxStatus `json:"status"`
	ErrorMsg    string   `json:"error_msg"`
	CreatedAt   int64    `json:"created_at"`
}

type FundTx struct {
	Key             string
	BatchID         string
	PubKey          string
	Msat            int64
	FeeMsat         int64
	DustMsat        int64
	PR              string
	PaymentHash     string
	PaymentPreimage string
	Status          TxStatus
	CreatedAt       int64
}
