package main

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	if err := initSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}
	return db, nil
}

func initSchema(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS vouchers (
		id                    INTEGER PRIMARY KEY,
		secret                TEXT NOT NULL,
		pubkey                TEXT NOT NULL,
		refund_code	          TEXT,
		refund_after_seconds  INTEGER NOT NULL,
		balance_msat          INTEGER NOT NULL DEFAULT 0,
		active                INTEGER NOT NULL DEFAULT 1,
		single_withdrawal     INTEGER NOT NULL DEFAULT 0,
		last_tx_at            TEXT,
		created_at            TEXT NOT NULL DEFAULT (datetime('now'))
	)`)
	return err
}

func insertVoucher(db *sql.DB, v *Voucher) error {
	result, err := db.Exec(
		`INSERT INTO vouchers (secret, pubkey, refund_code, refund_after_seconds, balance_msat, active, single_withdrawal)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		v.Secret, v.PubKey, nullableString(v.RefundCode), v.RefundAfterSeconds, v.BalanceMsat, boolToInt(v.Active), boolToInt(v.SingleWithdrawal),
	)
	if err != nil {
		return err
	}
	v.ID, err = result.LastInsertId()
	return err
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
