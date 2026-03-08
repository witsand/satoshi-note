package main

type Voucher struct {
	ID                 int64   `json:"id"`
	Secret             string  `json:"secret"`
	PubKey             string  `json:"pubkey"`
	RefundCode         string  `json:"refund_code,omitempty"`
	RefundAfterSeconds int     `json:"refund_after_seconds"`
	BalanceMsat        int64   `json:"balance_msat"`
	Active             bool    `json:"active"`
	SingleWithdrawal   bool    `json:"single_withdrawal"`
	LastTxAt           *string `json:"last_tx_at"`
}
