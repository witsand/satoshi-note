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
	ID                 int64
	Secret             string `json:"secret,omitempty"`
	ClaimLNURL         string `json:"claim_lnurl,omitempty"`
	PubKey             string `json:"pubkey,omitempty"`
	FundLNURL          string `json:"fund_lnurl,omitempty"`
	BatchName          string `json:"batch_name,omitempty"`
	BatchID            string `json:"batch_id,omitempty"`
	BatchFundLNURL     string `json:"batch_fund_lnurl,omitempty"`
	RefundCode         string `json:"refund_code,omitempty"`
	RefundAfterSeconds int64  `json:"refund_after_seconds"`
	BalanceMsat        int64  `json:"balance_msat,omitempty"`
	Active             bool   `json:"active,omitempty"`
	SingleUse          bool   `json:"single_use,omitempty"`
	Refunded           bool   `json:"-"`
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
