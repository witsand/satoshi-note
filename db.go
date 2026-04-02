package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	// Single writer prevents SQLITE_BUSY under concurrent HTTP handlers.
	db.SetMaxOpenConns(1)

	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		// "PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			db.Close()
			return nil, fmt.Errorf("pragma %q: %w", p, err)
		}
	}

	if err := initSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}

	if err := migrateSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate schema: %w", err)
	}

	return db, nil
}

func migrateSchema(db *sql.DB) error {
	migrations := []string{
		`ALTER TABLE vouchers ADD COLUMN transfers_only     INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE vouchers ADD COLUMN max_redeem_msat    INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE vouchers ADD COLUMN unique_redemptions INTEGER NOT NULL DEFAULT 0`,
	}
	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			if !strings.Contains(err.Error(), "duplicate column name") {
				return fmt.Errorf("%s: %w", m, err)
			}
		}
	}
	return nil
}

func initSchema(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS vouchers (
			id                    INTEGER PRIMARY KEY,
			pub_key               TEXT NOT NULL,
			batch_id              TEXT NOT NULL,
			refund_code           TEXT NOT NULL,
			refund_after_seconds  INTEGER NOT NULL,
			balance_msat          INTEGER NOT NULL DEFAULT 0,
			active                INTEGER NOT NULL DEFAULT 1,
			single_use            INTEGER NOT NULL,
			transfers_only        INTEGER NOT NULL DEFAULT 0,
			max_redeem_msat       INTEGER NOT NULL DEFAULT 0,
			unique_redemptions    INTEGER NOT NULL DEFAULT 0,
			refunded              INTEGER NOT NULL DEFAULT 0,
			refund_tx_id          INTEGER NOT NULL DEFAULT 0,
			created_at            INTEGER NOT NULL,
			updated_at            INTEGER NOT NULL DEFAULT 0
		)`,

		`CREATE TABLE IF NOT EXISTS fund_txs (
			key              TEXT PRIMARY KEY,
			pub_key          TEXT NOT NULL,
			batch_id         TEXT NOT NULL,
			msat             INTEGER NOT NULL,
			fee_msat         INTEGER NOT NULL,
			dust_msat        INTEGER NOT NULL DEFAULT 0,
			pr               TEXT NOT NULL,
			payment_hash     TEXT NOT NULL DEFAULT "",
			payment_preimage TEXT NOT NULL DEFAULT "",
			status           TEXT NOT NULL,
			created_at       INTEGER NOT NULL,
			updated_at       INTEGER NOT NULL DEFAULT 0
		)`,

		`CREATE TABLE IF NOT EXISTS refund_txs (
			id            INTEGER PRIMARY KEY,
			refund_code   TEXT    NOT NULL,
			amount_msat   INTEGER NOT NULL,
			db_tx_fee     INTEGER NOT NULL DEFAULT 0,
			actual_fee    INTEGER NOT NULL DEFAULT 0,
			refunded      INTEGER NOT NULL DEFAULT 0,
			error_msg     TEXT    NOT NULL DEFAULT "",
			created_at    INTEGER NOT NULL,
			updated_at    INTEGER NOT NULL DEFAULT 0
		)`,

		`CREATE TABLE IF NOT EXISTS redeem_sessions (
			k1         TEXT PRIMARY KEY,
			pub_key    TEXT NOT NULL,
			used       INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL DEFAULT 0
		)`,

		`CREATE TABLE IF NOT EXISTS redeem_txs (
			id            INTEGER PRIMARY KEY,
			voucher_id    INTEGER NOT NULL,
			secret        TEXT    NOT NULL DEFAULT "",
			pr            TEXT NOT NULL,
			msat          INTEGER NOT NULL,
			ln_fee        INTEGER NOT NULL,
			db_tx_fee     INTEGER NOT NULL DEFAULT 0,
			status        TEXT NOT NULL,
			actual_ln_fee INTEGER NOT NULL DEFAULT 0,
			created_at    INTEGER NOT NULL,
			updated_at    INTEGER NOT NULL DEFAULT 0,
			error_msg     TEXT    NOT NULL DEFAULT ""
		)`,

		`CREATE TABLE IF NOT EXISTS transfer_txs (
			id           INTEGER PRIMARY KEY,
			from_pub_key TEXT    NOT NULL,
			to_pub_key   TEXT    NOT NULL DEFAULT "",
			to_batch_id  TEXT    NOT NULL DEFAULT "",
			amount_msat  INTEGER NOT NULL,
			fee_msat     INTEGER NOT NULL,
			net_msat     INTEGER NOT NULL,
			dust_msat    INTEGER NOT NULL DEFAULT 0,
			created_at   INTEGER NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS redemption_fingerprints (
			voucher_id  INTEGER NOT NULL,
			fingerprint TEXT    NOT NULL,
			created_at  INTEGER NOT NULL,
			PRIMARY KEY (voucher_id, fingerprint)
		)`,

		`CREATE UNIQUE INDEX IF NOT EXISTS idx_vouchers_pub_key ON vouchers(pub_key)`,
		`CREATE INDEX IF NOT EXISTS idx_vouchers_batch_id       ON vouchers(batch_id)`,
		`CREATE INDEX IF NOT EXISTS idx_fund_txs_status         ON fund_txs(status)`,
		`CREATE INDEX IF NOT EXISTS idx_redeem_sessions_k1      ON redeem_sessions(k1, pub_key)`,
		`CREATE INDEX IF NOT EXISTS idx_redeem_txs_voucher_id   ON redeem_txs(voucher_id)`,
		`CREATE INDEX IF NOT EXISTS idx_refund_txs_refunded     ON refund_txs(refunded)`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("exec schema: %w", err)
		}
	}

	return nil
}

func (srv *Server) insertVoucher(v *Voucher) error {
	_, err := srv.db.Exec(
		`INSERT INTO vouchers (pub_key, batch_id, refund_code, refund_after_seconds, single_use, transfers_only, max_redeem_msat, unique_redemptions, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		v.PubKey, v.BatchID, v.RefundCode, v.RefundAfterSeconds,
		boolToInt(v.SingleUse), boolToInt(v.TransfersOnly), v.MaxRedeemMsat, boolToInt(v.UniqueRedemptions),
		time.Now().Unix(),
	)
	return err
}

func (srv *Server) insertRedeemSession(k1, pubKey string) error {
	_, err := srv.db.Exec(
		`INSERT INTO redeem_sessions (k1, pub_key, created_at) VALUES (?, ?, ?)`,
		k1, pubKey, time.Now().Unix(),
	)
	return err
}

func (srv *Server) insertRedeemTx(dbTx *sql.Tx, voucherID int64, secret, pr string, msat, fee int64) (int64, error) {
	res, err := dbTx.Exec(
		`INSERT INTO redeem_txs (voucher_id, secret, pr, msat, ln_fee, status, created_at) VALUES(?, ?, ?, ?, ?, ?, ?)`,
		voucherID, secret, pr, msat, fee, TxPending, time.Now().Unix(),
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

func (srv *Server) markRedeemSessionUsed(k1, pubKey string) error {
	res, err := srv.db.Exec(
		`UPDATE redeem_sessions SET used = 1, updated_at = ? WHERE k1 = ? AND pub_key = ? AND used = 0 AND created_at >= ?`,
		time.Now().Unix(), k1, pubKey, time.Now().Unix()-1800) // 30 minute window
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

// insertRedemptionFingerprint records a fingerprint for a voucher inside an existing transaction.
// Returns (true, nil) when the fingerprint is new, (false, nil) when it already existed (duplicate).
func insertRedemptionFingerprint(db dbQuerier, voucherID int64, fingerprint string) (bool, error) {
	res, err := db.Exec(
		`INSERT OR IGNORE INTO redemption_fingerprints (voucher_id, fingerprint, created_at) VALUES (?, ?, ?)`,
		voucherID, fingerprint, time.Now().Unix(),
	)
	if err != nil {
		return false, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

// usedFingerprints returns a set of voucher IDs for which the given fingerprint
// already exists in redemption_fingerprints.
func (srv *Server) usedFingerprints(voucherIDs []int64, fingerprint string) (map[int64]bool, error) {
	if len(voucherIDs) == 0 {
		return map[int64]bool{}, nil
	}
	placeholders := make([]string, len(voucherIDs))
	args := make([]any, len(voucherIDs)+1)
	args[0] = fingerprint
	for i, id := range voucherIDs {
		placeholders[i] = "?"
		args[i+1] = id
	}
	rows, err := srv.db.Query(
		`SELECT voucher_id FROM redemption_fingerprints WHERE fingerprint = ? AND voucher_id IN (`+strings.Join(placeholders, ",")+`)`,
		args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[int64]bool)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		result[id] = true
	}
	return result, rows.Err()
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
	keyLen := len(tx.PubKey) / 2
	if keyLen == 0 {
		keyLen = len(tx.BatchID) / 2
	}
	keyBytes := make([]byte, keyLen)
	if _, err := rand.Read(keyBytes); err != nil {
		return err
	}
	tx.Key = hex.EncodeToString(keyBytes)

	_, err := srv.db.Exec(
		`INSERT INTO fund_txs (key, batch_id, pub_key, msat, fee_msat, pr, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		tx.Key, tx.BatchID, tx.PubKey, tx.Msat, tx.FeeMsat, tx.PR, TxPending, time.Now().Unix(),
	)
	return err
}

func updateFundTXStatus(dbTx *sql.Tx, key string, status TxStatus, paymentHash, paymentPreimage string) error {
	res, err := dbTx.Exec(`
		UPDATE fund_txs SET status = ?, payment_hash = ?, payment_preimage = ?, updated_at = ?
		WHERE key = ? AND status = ?`,
		status, paymentHash, paymentPreimage, time.Now().Unix(), key, TxPending)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("fund tx %s already confirmed or not found", key)
	}
	return nil
}

func updateFundTxDust(dbTx *sql.Tx, key string, dustMsat int64) error {
	_, err := dbTx.Exec(`UPDATE fund_txs SET dust_msat = ? WHERE key = ?`, dustMsat, key)
	return err
}

func (srv *Server) getVoucherByPubKey(db dbQuerier, pubkey string) (*Voucher, error) {
	row := db.QueryRow(`SELECT id, refund_after_seconds, balance_msat, single_use, transfers_only, max_redeem_msat, unique_redemptions, updated_at
		FROM vouchers WHERE pub_key = ? AND active = 1`, pubkey)

	v := &Voucher{PubKey: pubkey}
	var updatedAt int64
	var singleUseInt, transfersOnlyInt, uniqueRedemptionsInt int
	if err := row.Scan(&v.ID, &v.RefundAfterSeconds, &v.BalanceMsat, &singleUseInt, &transfersOnlyInt, &v.MaxRedeemMsat, &uniqueRedemptionsInt, &updatedAt); err != nil {
		return nil, err
	}
	v.SingleUse = singleUseInt == 1
	v.TransfersOnly = transfersOnlyInt == 1
	v.UniqueRedemptions = uniqueRedemptionsInt == 1

	if time.Unix(updatedAt, 0).Add(time.Duration(v.RefundAfterSeconds)*time.Second).Before(time.Now()) && updatedAt != 0 {
		return nil, fmt.Errorf("voucher expired")
	}

	return v, nil
}

type voucherStatus struct {
	ID                int64
	BalanceMsat       int64
	ExpiresAt         int64 // updated_at + refund_after_seconds; 0 means no expiry clock started yet
	Active            bool
	Refunded          bool
	RefundPending     bool // refund tx allocated but not yet paid
	TransfersOnly     bool
	MaxRedeemMsat     int64
	UniqueRedemptions bool
}

func (srv *Server) getVoucherStatusBatch(pubKeys []string) (map[string]*voucherStatus, error) {
	if len(pubKeys) == 0 {
		return map[string]*voucherStatus{}, nil
	}
	placeholders := make([]string, len(pubKeys))
	args := make([]any, len(pubKeys))
	for i, pk := range pubKeys {
		placeholders[i] = "?"
		args[i] = pk
	}
	rows, err := srv.db.Query(
		`SELECT id, pub_key, balance_msat, updated_at, refund_after_seconds, active, refunded, refund_tx_id,
		        transfers_only, max_redeem_msat, unique_redemptions
		 FROM vouchers WHERE pub_key IN (`+strings.Join(placeholders, ",")+`)`,
		args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]*voucherStatus, len(pubKeys))
	for rows.Next() {
		var pubKey string
		var s voucherStatus
		var updatedAt, refundAfterSeconds, refundTxID int64
		var activeInt, refundedInt, transfersOnlyInt, uniqueRedemptionsInt int
		if err := rows.Scan(&s.ID, &pubKey, &s.BalanceMsat, &updatedAt, &refundAfterSeconds, &activeInt, &refundedInt, &refundTxID,
			&transfersOnlyInt, &s.MaxRedeemMsat, &uniqueRedemptionsInt); err != nil {
			return nil, err
		}
		s.Active = activeInt == 1
		s.Refunded = refundedInt == 1
		s.RefundPending = refundTxID > 0 && !s.Refunded
		s.TransfersOnly = transfersOnlyInt == 1
		s.UniqueRedemptions = uniqueRedemptionsInt == 1
		if updatedAt != 0 {
			s.ExpiresAt = updatedAt + refundAfterSeconds
		}
		result[pubKey] = &s
	}
	return result, rows.Err()
}

func (srv *Server) getVouchersByBatchID(db dbQuerier, batchID string) ([]Voucher, error) {
	rows, err := db.Query(`SELECT id, refund_after_seconds, balance_msat, updated_at
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
		if err := rows.Scan(&item.v.ID, &item.v.RefundAfterSeconds, &item.v.BalanceMsat, &item.updatedAt); err != nil {
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

func (srv *Server) insertRefundTx(dbTx *sql.Tx, refundCode string, amountMsat, dbTxFee int64) (int64, error) {
	res, err := dbTx.Exec(
		`INSERT INTO refund_txs (refund_code, amount_msat, db_tx_fee, created_at)
		 VALUES (?, ?, ?, ?)`,
		refundCode, amountMsat, dbTxFee, time.Now().Unix(),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (srv *Server) markVouchersRefunded(dbTx *sql.Tx, ids []int64, refundTxID int64) error {
	for _, id := range ids {
		_, err := dbTx.Exec(
			`UPDATE vouchers SET balance_msat = 0, refunded = 1, active = 0, refund_tx_id = ?, updated_at = ? WHERE id = ?`,
			refundTxID, time.Now().Unix(), id,
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

func (srv *Server) markRefundTxPaid(id, netDbTxFee, actualFee int64) error {
	_, err := srv.db.Exec(
		`UPDATE refund_txs SET refunded = 1, actual_fee = ?, db_tx_fee = ?, updated_at = ? WHERE id = ?`,
		actualFee, netDbTxFee, time.Now().Unix(), id,
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

type LedgerStats struct {
	VouchersBalanceMsat      int64   `json:"vouchers_balance_msat"`
	FundTxsDustMsat          int64   `json:"fund_txs_dust_msat"`
	RefundTxsDbTxFee         int64   `json:"refund_txs_db_tx_fee"`
	RefundTxsPendingMsat     int64   `json:"refund_txs_pending_msat"`
	RedeemTxsDbTxFee         int64   `json:"redeem_txs_db_tx_fee"`
	TransferTxsFeeMsat       int64   `json:"transfer_txs_fee_msat"`
	TransferTxsDustMsat      int64   `json:"transfer_txs_dust_msat"`
	VouchersAvgSecsToExpiry  float64 `json:"vouchers_avg_secs_to_expiry"`
	VouchersWithBalanceCount int64   `json:"vouchers_with_balance_count"`
}

func (srv *Server) getLedgerStats() (LedgerStats, error) {
	row := srv.db.QueryRow(`
		SELECT
			(SELECT COALESCE(SUM(balance_msat), 0) FROM vouchers WHERE balance_msat > 0),
			(SELECT COALESCE(SUM(dust_msat),    0) FROM fund_txs),
			(SELECT COALESCE(SUM(db_tx_fee),    0) FROM refund_txs),
			(SELECT COALESCE(SUM(amount_msat),  0) FROM refund_txs WHERE refunded = 0),
			(SELECT COALESCE(SUM(db_tx_fee),    0) FROM redeem_txs),
			(SELECT COALESCE(SUM(fee_msat),     0) FROM transfer_txs),
			(SELECT COALESCE(SUM(dust_msat),    0) FROM transfer_txs),
			(SELECT COALESCE(
				CAST(SUM(balance_msat * (created_at + refund_after_seconds - ?)) AS REAL)
				/ NULLIF(SUM(balance_msat), 0),
				0
			) FROM vouchers WHERE balance_msat > 0),
			(SELECT COUNT(*) FROM vouchers WHERE balance_msat > 0)
	`, time.Now().Unix())
	var s LedgerStats
	err := row.Scan(
		&s.VouchersBalanceMsat,
		&s.FundTxsDustMsat,
		&s.RefundTxsDbTxFee,
		&s.RefundTxsPendingMsat,
		&s.RedeemTxsDbTxFee,
		&s.TransferTxsFeeMsat,
		&s.TransferTxsDustMsat,
		&s.VouchersAvgSecsToExpiry,
		&s.VouchersWithBalanceCount,
	)
	return s, err
}

func (srv *Server) insertTransferTx(dbTx *sql.Tx, fromPubKey, toPubKey, toBatchID string, amountMsat, feeMsat, netMsat, dustMsat int64) error {
	_, err := dbTx.Exec(
		`INSERT INTO transfer_txs (from_pub_key, to_pub_key, to_batch_id, amount_msat, fee_msat, net_msat, dust_msat, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		fromPubKey, toPubKey, toBatchID, amountMsat, feeMsat, netMsat, dustMsat, time.Now().Unix(),
	)
	return err
}
