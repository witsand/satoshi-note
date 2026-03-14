package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// dbQuerier is satisfied by both *sql.DB and *sql.Tx, allowing query functions
// to operate inside or outside a transaction.
type dbQuerier interface {
	QueryRow(query string, args ...any) *sql.Row
	Query(query string, args ...any) (*sql.Rows, error)
	Exec(query string, args ...any) (sql.Result, error)
}

func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	// SQLite requires a single writer connection to avoid "database is locked" errors.
	db.SetMaxOpenConns(1)

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable WAL: %w", err)
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
		pub_key               TEXT NOT NULL,
		batch_name            TEXT NOT NULL,
		batch_id              TEXT NOT NULL,
		refund_code           TEXT NOT NULL,
		refund_after_seconds  INTEGER NOT NULL,
		balance_msat          INTEGER NOT NULL DEFAULT 0,
		active                INTEGER NOT NULL DEFAULT 1,
		single_use            INTEGER NOT NULL,
		refunded              INTEGER NOT NULL DEFAULT 0,
		created_at            INTEGER NOT NULL,
		updated_at            INTEGER NOT NULL DEFAULT 0
	)`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS fund_txs (
		key              TEXT PRIMARY KEY,
		batch_id         TEXT NOT NULL,
		pub_key          TEXT NOT NULL,
		msat             INTEGER NOT NULL,
		fee_msat         INTEGER NOT NULL,
		pr               TEXT NOT NULL,
		payment_hash     TEXT NOT NULL DEFAULT "",
		payment_preimage TEXT NOT NULL DEFAULT "",
		status           TEXT NOT NULL,
		created_at       INTEGER NOT NULL,
		updated_at       INTEGER NOT NULL DEFAULT 0
	)`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS refund_txs (
		id            INTEGER PRIMARY KEY,
		refund_code   TEXT    NOT NULL,
		amount_msat   INTEGER NOT NULL,
		db_tx_fee     INTEGER NOT NULL DEFAULT 0,
		actual_fee    INTEGER NOT NULL DEFAULT 0,
		refunded      INTEGER NOT NULL DEFAULT 0,
		error_msg     TEXT    NOT NULL DEFAULT "",
		created_at    INTEGER NOT NULL,
		updated_at    INTEGER NOT NULL DEFAULT 0
	)`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS redeem_sessions (
		k1         TEXT PRIMARY KEY,
		secret     TEXT NOT NULL,
		used       INTEGER NOT NULL DEFAULT 0,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL DEFAULT 0
	)`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS redeem_txs (
		id            INTEGER PRIMARY KEY,
		voucher_id    INTEGER NOT NULL,
		pr            TEXT NOT NULL,
		msat          INTEGER NOT NULL,
		ln_fee        INTEGER NOT NULL,
		db_tx_fee     INTEGER NOT NULL DEFAULT 0,
		status        TEXT NOT NULL,
		actual_ln_fee INTEGER NOT NULL DEFAULT 0,
		created_at    INTEGER NOT NULL,
		updated_at    INTEGER NOT NULL DEFAULT 0,
		error_msg     TEXT    NOT NULL DEFAULT ""
	)`)
	if err != nil {
		return err
	}

	// Indexes for frequently queried columns.
	indexes := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_vouchers_secret  ON vouchers(secret)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_vouchers_pub_key ON vouchers(pub_key)`,
		`CREATE INDEX IF NOT EXISTS idx_vouchers_batch_id       ON vouchers(batch_id)`,
		`CREATE INDEX IF NOT EXISTS idx_fund_txs_status         ON fund_txs(status)`,
		`CREATE INDEX IF NOT EXISTS idx_redeem_sessions_k1      ON redeem_sessions(k1, secret)`,
		`CREATE INDEX IF NOT EXISTS idx_redeem_txs_voucher_id   ON redeem_txs(voucher_id)`,
		`CREATE INDEX IF NOT EXISTS idx_refund_txs_refunded     ON refund_txs(refunded)`,
	}
	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			return err
		}
	}

	return nil
}

func (srv *Server) insertVoucher(v *Voucher) error {
	_, err := srv.db.Exec(
		`INSERT INTO vouchers (secret, pub_key, batch_name, batch_id, refund_code, refund_after_seconds, single_use, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		v.Secret, v.PubKey, v.BatchName, v.BatchID, v.RefundCode, v.RefundAfterSeconds, boolToInt(v.SingleUse), time.Now().Unix(),
	)
	return err
}

func (srv *Server) insertRedeemSession(k1, secret string) error {
	_, err := srv.db.Exec(
		`INSERT INTO redeem_sessions (k1, secret, created_at) VALUES (?, ?, ?)`,
		k1, secret, time.Now().Unix(),
	)
	return err
}

func (srv *Server) insertRedeemTx(dbTx *sql.Tx, voucherID int64, pr string, msat, fee int64) (int64, error) {
	res, err := dbTx.Exec(
		`INSERT INTO redeem_txs (voucher_id, pr, msat, ln_fee, status, created_at) VALUES(?, ?, ?, ?, ?, ?)`,
		voucherID, pr, msat, fee, TxPending, time.Now().Unix(),
	)
	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

func (srv *Server) updateRedeemTx(redeemID int64, status TxStatus, dbTxFee, actualLNFee int64, errMsg string) error {
	_, err := srv.db.Exec(
		`UPDATE redeem_txs SET status = ?, db_tx_fee = ?, actual_ln_fee = ?, error_msg = ?, updated_at = ? WHERE id = ?`,
		status, dbTxFee, actualLNFee, errMsg, time.Now().Unix(), redeemID,
	)
	return err
}

func (srv *Server) markRedeemSessionUsed(k1, secret string) error {
	res, err := srv.db.Exec(
		`UPDATE redeem_sessions SET used = 1, updated_at = ? WHERE k1 = ? AND secret = ? AND used = 0 AND created_at >= ?`,
		time.Now().Unix(), k1, secret, time.Now().Unix()-1800) // 30 minute window
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("redeem session not found, expired or already used")
	}

	return nil
}

func (srv *Server) updateVoucherBalance(dbTx *sql.Tx, id int64, msats int64) error {
	res, err := dbTx.Exec(`
		UPDATE vouchers
		SET
			balance_msat = balance_msat + ?,
			active = CASE
				WHEN ? < 0 AND single_use = 1 THEN 0
				ELSE active
			END,
			updated_at = ?
		WHERE id = ? AND balance_msat >= -?`, msats, msats, time.Now().Unix(), id, msats)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("voucher not found or balance too low")
	}

	return nil
}

// addVoucherBalance unconditionally adds msats to a voucher balance (used for restoring
// balance after a failed payment). Does not check single_use or minimum balance.
func (srv *Server) addVoucherBalance(id int64, msats int64) error {
	_, err := srv.db.Exec(
		`UPDATE vouchers SET balance_msat = balance_msat + ?, active = 1, updated_at = ? WHERE id = ?`,
		msats, time.Now().Unix(), id,
	)
	return err
}

func (srv *Server) insertFundTX(tx *FundTx) error {
	batchIDBytes := make([]byte, srv.cfg.randomBytesLength)
	if _, err := rand.Read(batchIDBytes); err != nil {
		return err
	}
	tx.Key = hex.EncodeToString(batchIDBytes)

	_, err := srv.db.Exec(
		`INSERT INTO fund_txs (key, batch_id, pub_key, msat, fee_msat, pr, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		tx.Key, tx.BatchID, tx.PubKey, tx.Msat, tx.FeeMsat, tx.PR, TxPending, time.Now().Unix(),
	)
	return err
}

func updateFundTXStatus(dbTx *sql.Tx, key string, status TxStatus, paymentHash, paymentPreimage string) error {
	_, err := dbTx.Exec(`
		UPDATE fund_txs SET status = ?, payment_hash = ?, payment_preimage = ?, updated_at = ?
		WHERE key = ? AND status = ?`,
		status, paymentHash, paymentPreimage, time.Now().Unix(), key, TxPending)
	return err
}

func (srv *Server) getVoucherByPubKey(db dbQuerier, pubkey string) (*Voucher, error) {
	row := db.QueryRow(`SELECT id, refund_after_seconds, balance_msat, updated_at
		FROM vouchers WHERE pub_key = ? AND active = 1`, pubkey)

	v := &Voucher{PubKey: pubkey}
	var updatedAt int64
	if err := row.Scan(&v.ID, &v.RefundAfterSeconds, &v.BalanceMsat, &updatedAt); err != nil {
		return nil, err
	}

	if time.Unix(updatedAt, 0).Add(time.Duration(v.RefundAfterSeconds)*time.Second).Before(time.Now()) && updatedAt != 0 {
		srv.deactivateVoucher(v.ID)
		return nil, fmt.Errorf("voucher expired")
	}

	return v, nil
}

func (srv *Server) getVoucherBySecret(db dbQuerier, secret string) (*Voucher, error) {
	row := db.QueryRow(`SELECT id, refund_after_seconds, balance_msat, single_use, updated_at
		FROM vouchers WHERE secret = ? AND active = 1`, secret)

	v := &Voucher{Secret: secret}
	var updatedAt int64
	if err := row.Scan(&v.ID, &v.RefundAfterSeconds, &v.BalanceMsat, &v.SingleUse, &updatedAt); err != nil {
		return nil, err
	}

	if time.Unix(updatedAt, 0).Add(time.Duration(v.RefundAfterSeconds)*time.Second).Before(time.Now()) && updatedAt != 0 {
		srv.deactivateVoucher(v.ID)
		return nil, fmt.Errorf("voucher expired")
	}

	return v, nil
}

func (srv *Server) getVouchersByBatchID(db dbQuerier, batchID string) ([]Voucher, error) {
	rows, err := db.Query(`SELECT id, batch_name, refund_after_seconds, balance_msat, updated_at
		FROM vouchers WHERE batch_id = ? AND active = 1`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type voucherRow struct {
		v         Voucher
		updatedAt int64
	}

	var items []voucherRow
	for rows.Next() {
		var item voucherRow
		item.v.BatchID = batchID
		if err := rows.Scan(&item.v.ID, &item.v.BatchName, &item.v.RefundAfterSeconds, &item.v.BalanceMsat, &item.updatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var vs []Voucher
	for _, item := range items {
		if time.Unix(item.updatedAt, 0).Add(time.Duration(item.v.RefundAfterSeconds)*time.Second).Before(time.Now()) && item.updatedAt != 0 {
			srv.deactivateVoucher(item.v.ID)
			continue
		}
		vs = append(vs, item.v)
	}

	if len(vs) == 0 {
		return nil, fmt.Errorf("all vouchers in batch have expired")
	}

	return vs, nil
}

func (srv *Server) getPendingFundTxs() ([]FundTx, error) {
	rows, err := srv.db.Query("SELECT key, batch_id, pub_key, msat, fee_msat, pr, created_at FROM fund_txs WHERE status = ?", TxPending)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []FundTx
	for rows.Next() {
		tx := FundTx{}
		if err := rows.Scan(&tx.Key, &tx.BatchID, &tx.PubKey, &tx.Msat, &tx.FeeMsat, &tx.PR, &tx.CreatedAt); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}

	return txs, rows.Err()
}

func (srv *Server) getFundTxByKey(key string) (*FundTx, error) {
	row := srv.db.QueryRow("SELECT pr, payment_hash, payment_preimage, status FROM fund_txs WHERE key = ?", key)

	tx := &FundTx{Key: key}
	if err := row.Scan(&tx.PR, &tx.PaymentHash, &tx.PaymentPreimage, &tx.Status); err != nil {
		return nil, err
	}

	return tx, nil
}

func (srv *Server) getFundTxByPR(pr string) (*FundTx, error) {
	row := srv.db.QueryRow("SELECT key, batch_id, pub_key, msat, fee_msat, payment_hash, payment_preimage, status FROM fund_txs WHERE pr = ? AND status = ?", pr, TxPending)

	tx := &FundTx{PR: pr}
	if err := row.Scan(&tx.Key, &tx.BatchID, &tx.PubKey, &tx.Msat, &tx.FeeMsat, &tx.PaymentHash, &tx.PaymentPreimage, &tx.Status); err != nil {
		return nil, err
	}

	return tx, nil
}

func (srv *Server) deactivateVoucher(id int64) error {
	_, err := srv.db.Exec(`UPDATE vouchers SET active = 0, updated_at = ? WHERE id = ?`, time.Now().Unix(), id)
	return err
}

func (srv *Server) getExpiredVouchersWithBalance() ([]Voucher, error) {
	rows, err := srv.db.Query(`
		SELECT id, refund_code, balance_msat
		FROM vouchers
		WHERE active = 1
		  AND balance_msat > 0
		  AND refunded = 0
		  AND updated_at > 0
		  AND (updated_at + refund_after_seconds) <= ?`,
		time.Now().Unix(),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vs []Voucher
	for rows.Next() {
		var v Voucher
		if err := rows.Scan(&v.ID, &v.RefundCode, &v.BalanceMsat); err != nil {
			return nil, err
		}
		vs = append(vs, v)
	}
	return vs, rows.Err()
}

func (srv *Server) insertRefundTx(dbTx *sql.Tx, refundCode string, amountMsat, dbTxFee int64, refunded bool, errorMsg string) (int64, error) {
	res, err := dbTx.Exec(
		`INSERT INTO refund_txs (refund_code, amount_msat, db_tx_fee, refunded, error_msg, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		refundCode, amountMsat, dbTxFee, boolToInt(refunded), errorMsg, time.Now().Unix(),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (srv *Server) markVouchersRefunded(dbTx *sql.Tx, ids []int64) error {
	for _, id := range ids {
		_, err := dbTx.Exec(
			`UPDATE vouchers SET balance_msat = 0, refunded = 1, active = 0, updated_at = ? WHERE id = ?`,
			time.Now().Unix(), id,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (srv *Server) getPendingRefundTxs() ([]RefundTx, error) {
	rows, err := srv.db.Query(
		`SELECT id, refund_code, amount_msat, db_tx_fee FROM refund_txs WHERE refunded = 0`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []RefundTx
	for rows.Next() {
		var rt RefundTx
		if err := rows.Scan(&rt.ID, &rt.RefundCode, &rt.AmountMsat, &rt.DbTxFee); err != nil {
			return nil, err
		}
		txs = append(txs, rt)
	}
	return txs, rows.Err()
}

func (srv *Server) markRefundTxPaid(id, actualFee int64) error {
	_, err := srv.db.Exec(
		`UPDATE refund_txs SET refunded = 1, actual_fee = ?, updated_at = ? WHERE id = ?`,
		actualFee, time.Now().Unix(), id,
	)
	return err
}

func (srv *Server) markRefundTxFailed(id int64, errMsg string) error {
	_, err := srv.db.Exec(
		`UPDATE refund_txs SET error_msg = ?, updated_at = ? WHERE id = ?`,
		errMsg, time.Now().Unix(), id,
	)
	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
