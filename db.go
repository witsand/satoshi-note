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

	if err := migrateFundKeys(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate fund keys: %w", err)
	}

	return db, nil
}

func migrateSchema(db *sql.DB) error {
	migrations := []string{
		`ALTER TABLE vouchers ADD COLUMN transfers_only     INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE vouchers ADD COLUMN max_redeem_msat    INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE vouchers ADD COLUMN unique_redemptions INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE vouchers ADD COLUMN absolute_expiry    INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE vouchers ADD COLUMN regular_refund_first_at      INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE vouchers ADD COLUMN regular_refund_interval_secs INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE vouchers ADD COLUMN regular_refund_last_at       INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE vouchers ADD COLUMN fund_key TEXT NOT NULL DEFAULT ""`,
		// Migrate legacy single refund_code into the new voucher_refund_codes table.
		// The NOT EXISTS guard makes this idempotent across restarts.
		// If refund_code column is already dropped, SQLite returns "no such column" — ignored below.
		`INSERT INTO voucher_refund_codes (voucher_id, refund_code, share)
		 SELECT id, refund_code, 1 FROM vouchers
		 WHERE refund_code != ''
		   AND NOT EXISTS (
		     SELECT 1 FROM voucher_refund_codes vrc WHERE vrc.voucher_id = vouchers.id
		   )`,
		// Drop the now-redundant column. "no such column" means already dropped — ignored below.
		`ALTER TABLE vouchers DROP COLUMN refund_code`,
		// Invert refund relationship: track voucher_id on refund_txs instead of
		// refunded/refund_tx_id on vouchers.
		`ALTER TABLE refund_txs ADD COLUMN voucher_id INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE vouchers DROP COLUMN refunded`,
		`ALTER TABLE vouchers DROP COLUMN refund_tx_id`,
		`CREATE INDEX IF NOT EXISTS idx_refund_txs_voucher_id ON refund_txs(voucher_id)`,
	}
	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			errStr := err.Error()
			if !strings.Contains(errStr, "duplicate column name") &&
				!strings.Contains(errStr, "no such column") {
				return fmt.Errorf("%s: %w", m, err)
			}
		}
	}
	return nil
}

func migrateFundKeys(db *sql.DB) error {
	// Populate fund_key for any existing vouchers that predate this column.
	rows, err := db.Query(`SELECT id, pub_key FROM vouchers WHERE fund_key = ""`)
	if err != nil {
		return fmt.Errorf("query vouchers for fund_key migration: %w", err)
	}
	type row struct {
		id     int64
		pubKey string
	}
	var todo []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.pubKey); err != nil {
			rows.Close()
			return fmt.Errorf("scan voucher row: %w", err)
		}
		todo = append(todo, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, r := range todo {
		fundKey, err := secretToPubKey(r.pubKey)
		if err != nil {
			return fmt.Errorf("derive fund_key for voucher %d: %w", r.id, err)
		}
		if _, err := db.Exec(`UPDATE vouchers SET fund_key = ? WHERE id = ?`, fundKey, r.id); err != nil {
			return fmt.Errorf("update fund_key for voucher %d: %w", r.id, err)
		}
	}
	// Create unique index (idempotent).
	if _, err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_vouchers_fund_key ON vouchers(fund_key)`); err != nil {
		return fmt.Errorf("create fund_key index: %w", err)
	}
	return nil
}

func initSchema(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS vouchers (
			id                             INTEGER PRIMARY KEY,
			pub_key                        TEXT    NOT NULL,
			fund_key                       TEXT    NOT NULL DEFAULT "",
			batch_id                       TEXT    NOT NULL,
			refund_after_seconds           INTEGER NOT NULL,
			absolute_expiry                INTEGER NOT NULL DEFAULT 0,
			balance_msat                   INTEGER NOT NULL DEFAULT 0,
			single_use                     INTEGER NOT NULL,
			transfers_only                 INTEGER NOT NULL DEFAULT 0,
			max_redeem_msat                INTEGER NOT NULL DEFAULT 0,
			unique_redemptions             INTEGER NOT NULL DEFAULT 0,
			regular_refund_first_at        INTEGER NOT NULL DEFAULT 0,
			regular_refund_interval_secs   INTEGER NOT NULL DEFAULT 0,
			regular_refund_last_at         INTEGER NOT NULL DEFAULT 0,
			active                         INTEGER NOT NULL DEFAULT 1,
			created_at                     INTEGER NOT NULL,
			updated_at                     INTEGER NOT NULL DEFAULT 0
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
			voucher_id    INTEGER NOT NULL DEFAULT 0,
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

		`CREATE TABLE IF NOT EXISTS voucher_refund_codes (
			id          INTEGER PRIMARY KEY,
			voucher_id  INTEGER NOT NULL,
			refund_code TEXT    NOT NULL,
			share       INTEGER NOT NULL DEFAULT 1
		)`,

		`CREATE UNIQUE INDEX IF NOT EXISTS idx_vouchers_pub_key ON vouchers(pub_key)`,
		`CREATE INDEX IF NOT EXISTS idx_vouchers_batch_id       ON vouchers(batch_id)`,
		`CREATE INDEX IF NOT EXISTS idx_fund_txs_status         ON fund_txs(status)`,
		`CREATE INDEX IF NOT EXISTS idx_redeem_sessions_k1      ON redeem_sessions(k1, pub_key)`,
		`CREATE INDEX IF NOT EXISTS idx_redeem_txs_voucher_id   ON redeem_txs(voucher_id)`,
		`CREATE INDEX IF NOT EXISTS idx_refund_txs_refunded     ON refund_txs(refunded)`,
		`CREATE INDEX IF NOT EXISTS idx_refund_txs_voucher_id   ON refund_txs(voucher_id)`,
		`CREATE INDEX IF NOT EXISTS idx_vrc_voucher_id          ON voucher_refund_codes(voucher_id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_vouchers_fund_key   ON vouchers(fund_key)`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("exec schema: %w", err)
		}
	}

	return nil
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
	row := db.QueryRow(`SELECT id, refund_after_seconds, balance_msat, single_use, transfers_only, max_redeem_msat, unique_redemptions, updated_at, created_at, absolute_expiry
		FROM vouchers WHERE pub_key = ? AND active = 1`, pubkey)

	v := &Voucher{PubKey: pubkey}
	var updatedAt, createdAt int64
	var singleUseInt, transfersOnlyInt, uniqueRedemptionsInt, absoluteExpiryInt int
	if err := row.Scan(&v.ID, &v.RefundAfterSeconds, &v.BalanceMsat, &singleUseInt, &transfersOnlyInt, &v.MaxRedeemMsat, &uniqueRedemptionsInt, &updatedAt, &createdAt, &absoluteExpiryInt); err != nil {
		return nil, err
	}
	v.SingleUse = singleUseInt == 1
	v.TransfersOnly = transfersOnlyInt == 1
	v.UniqueRedemptions = uniqueRedemptionsInt == 1
	v.AbsoluteExpiry = absoluteExpiryInt == 1

	var expired bool
	if v.AbsoluteExpiry {
		expired = time.Unix(createdAt, 0).Add(time.Duration(v.RefundAfterSeconds) * time.Second).Before(time.Now())
	} else {
		expired = updatedAt != 0 && time.Unix(updatedAt, 0).Add(time.Duration(v.RefundAfterSeconds)*time.Second).Before(time.Now())
	}
	if expired {
		return nil, fmt.Errorf("voucher expired")
	}

	return v, nil
}

func (srv *Server) getVoucherByFundKey(db dbQuerier, fundKey string) (*Voucher, error) {
	row := db.QueryRow(`SELECT id, pub_key, refund_after_seconds, balance_msat, single_use, transfers_only, max_redeem_msat, unique_redemptions, updated_at, created_at, absolute_expiry
		FROM vouchers WHERE fund_key = ? AND active = 1`, fundKey)

	v := &Voucher{FundKey: fundKey}
	var updatedAt, createdAt int64
	var singleUseInt, transfersOnlyInt, uniqueRedemptionsInt, absoluteExpiryInt int
	if err := row.Scan(&v.ID, &v.PubKey, &v.RefundAfterSeconds, &v.BalanceMsat, &singleUseInt, &transfersOnlyInt, &v.MaxRedeemMsat, &uniqueRedemptionsInt, &updatedAt, &createdAt, &absoluteExpiryInt); err != nil {
		return nil, err
	}
	v.SingleUse = singleUseInt == 1
	v.TransfersOnly = transfersOnlyInt == 1
	v.UniqueRedemptions = uniqueRedemptionsInt == 1
	v.AbsoluteExpiry = absoluteExpiryInt == 1

	var expired bool
	if v.AbsoluteExpiry {
		expired = time.Unix(createdAt, 0).Add(time.Duration(v.RefundAfterSeconds) * time.Second).Before(time.Now())
	} else {
		expired = updatedAt != 0 && time.Unix(updatedAt, 0).Add(time.Duration(v.RefundAfterSeconds)*time.Second).Before(time.Now())
	}
	if expired {
		return nil, fmt.Errorf("voucher expired")
	}

	return v, nil
}

type voucherStatus struct {
	ID                int64
	BalanceMsat       int64
	ExpiresAt         int64 // expiry epoch; 0 means no expiry clock started yet (relative-expiry only)
	Expired           bool  // true when ExpiresAt is in the past; may lead Active in the DB
	Active            bool
	Refunded          bool
	RefundPending     bool  // refund tx allocated but not yet paid
	LastRefundAt      int64 // unix timestamp of last successful refund payment; 0 if none
	TransfersOnly     bool
	MaxRedeemMsat     int64
	UniqueRedemptions bool
	AbsoluteExpiry    bool
	CreatedAt         int64
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
		`SELECT v.id, v.pub_key, v.balance_msat, v.updated_at, v.refund_after_seconds, v.active,
		        v.transfers_only, v.max_redeem_msat, v.unique_redemptions, v.created_at, v.absolute_expiry,
		        MAX(CASE WHEN rt.refunded = 1 THEN 1 ELSE 0 END)                              AS is_refunded,
		        MAX(CASE WHEN rt.refunded = 0 AND rt.error_msg = '' THEN 1 ELSE 0 END)        AS refund_pending,
		        COALESCE(MAX(CASE WHEN rt.refunded = 1 THEN rt.updated_at ELSE 0 END), 0)     AS last_refund_at
		 FROM vouchers v
		 LEFT JOIN refund_txs rt ON rt.voucher_id = v.id
		 WHERE v.pub_key IN (`+strings.Join(placeholders, ",")+`)
		 GROUP BY v.id`,
		args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]*voucherStatus, len(pubKeys))
	for rows.Next() {
		var pubKey string
		var s voucherStatus
		var updatedAt, refundAfterSeconds int64
		var activeInt, isRefundedInt, refundPendingInt, transfersOnlyInt, uniqueRedemptionsInt, absoluteExpiryInt int
		if err := rows.Scan(&s.ID, &pubKey, &s.BalanceMsat, &updatedAt, &refundAfterSeconds, &activeInt,
			&transfersOnlyInt, &s.MaxRedeemMsat, &uniqueRedemptionsInt, &s.CreatedAt, &absoluteExpiryInt,
			&isRefundedInt, &refundPendingInt, &s.LastRefundAt); err != nil {
			return nil, err
		}
		s.Active = activeInt == 1
		s.Refunded = isRefundedInt == 1
		s.RefundPending = refundPendingInt == 1 && !s.Refunded
		s.TransfersOnly = transfersOnlyInt == 1
		s.UniqueRedemptions = uniqueRedemptionsInt == 1
		s.AbsoluteExpiry = absoluteExpiryInt == 1
		if s.AbsoluteExpiry {
			s.ExpiresAt = s.CreatedAt + refundAfterSeconds
		} else if updatedAt != 0 {
			s.ExpiresAt = updatedAt + refundAfterSeconds
		}
		s.Expired = s.ExpiresAt > 0 && s.ExpiresAt <= time.Now().Unix()
		result[pubKey] = &s
	}
	return result, rows.Err()
}

func (srv *Server) getVouchersByBatchID(db dbQuerier, batchID string) ([]Voucher, error) {
	rows, err := db.Query(`SELECT id, refund_after_seconds, balance_msat, updated_at, created_at, absolute_expiry
		FROM vouchers WHERE batch_id = ? AND active = 1`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type voucherRow struct {
		v              Voucher
		updatedAt      int64
		createdAt      int64
		absoluteExpiry bool
	}

	var items []voucherRow
	for rows.Next() {
		var item voucherRow
		var absoluteExpiryInt int
		item.v.BatchID = batchID
		if err := rows.Scan(&item.v.ID, &item.v.RefundAfterSeconds, &item.v.BalanceMsat, &item.updatedAt, &item.createdAt, &absoluteExpiryInt); err != nil {
			return nil, err
		}
		item.absoluteExpiry = absoluteExpiryInt == 1
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var vs []Voucher
	for _, item := range items {
		var expired bool
		if item.absoluteExpiry {
			expired = time.Unix(item.createdAt, 0).Add(time.Duration(item.v.RefundAfterSeconds) * time.Second).Before(time.Now())
		} else {
			expired = item.updatedAt != 0 && time.Unix(item.updatedAt, 0).Add(time.Duration(item.v.RefundAfterSeconds)*time.Second).Before(time.Now())
		}
		if expired {
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
	now := time.Now().Unix()
	rows, err := srv.db.Query(`
		SELECT id, balance_msat
		FROM vouchers
		WHERE active = 1
		  AND balance_msat > 0
		  AND (
		    (absolute_expiry = 1 AND (created_at + refund_after_seconds) <= ?)
		    OR
		    (absolute_expiry = 0 AND updated_at > 0 AND (updated_at + refund_after_seconds) <= ?)
		  )`,
		now, now,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vs []Voucher
	for rows.Next() {
		var v Voucher
		if err := rows.Scan(&v.ID, &v.BalanceMsat); err != nil {
			return nil, err
		}
		vs = append(vs, v)
	}
	return vs, rows.Err()
}

type regularVoucher struct {
	ID           int64
	BalanceMsat  int64
	FirstAt      int64
	IntervalSecs int64
	LastAt       int64
}

func (srv *Server) getRegularRefundDueVouchers() ([]regularVoucher, error) {
	now := time.Now().Unix()
	rows, err := srv.db.Query(`
		SELECT id, balance_msat, regular_refund_first_at, regular_refund_interval_secs, regular_refund_last_at
		FROM vouchers
		WHERE active = 1
		  AND regular_refund_first_at > 0
		  AND (
		    (absolute_expiry = 1 AND (created_at + refund_after_seconds) > ?)
		    OR (absolute_expiry = 0 AND (updated_at = 0 OR (updated_at + refund_after_seconds) > ?))
		  )
		  AND (
		    (regular_refund_last_at > 0 AND (regular_refund_last_at + regular_refund_interval_secs) <= ?)
		    OR (regular_refund_last_at = 0 AND regular_refund_first_at <= ?)
		  )`,
		now, now, now, now,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vs []regularVoucher
	for rows.Next() {
		var v regularVoucher
		if err := rows.Scan(&v.ID, &v.BalanceMsat, &v.FirstAt, &v.IntervalSecs, &v.LastAt); err != nil {
			return nil, err
		}
		vs = append(vs, v)
	}
	return vs, rows.Err()
}

type regularRefundEntry struct {
	ID        int64
	NewLastAt int64
}

func advanceRegularRefundTime(db dbQuerier, entries []regularRefundEntry) error {
	for _, e := range entries {
		if _, err := db.Exec(
			`UPDATE vouchers SET regular_refund_last_at = ? WHERE id = ?`,
			e.NewLastAt, e.ID,
		); err != nil {
			return err
		}
	}
	return nil
}

func markVouchersRegularRefunded(dbTx *sql.Tx, entries []regularRefundEntry) error {
	for _, e := range entries {
		if _, err := dbTx.Exec(
			`UPDATE vouchers SET balance_msat = 0, regular_refund_last_at = ? WHERE id = ?`,
			e.NewLastAt, e.ID,
		); err != nil {
			return err
		}
	}
	return nil
}

func (srv *Server) nextRefundAt() (*time.Time, error) {
	now := time.Now().Unix()
	row := srv.db.QueryRow(`
		SELECT MIN(next_at) FROM (
		  SELECT CASE WHEN absolute_expiry = 1
		    THEN created_at + refund_after_seconds
		    ELSE updated_at + refund_after_seconds
		  END as next_at
		  FROM vouchers
		  WHERE active = 1 AND balance_msat > 0
		    AND (absolute_expiry = 1 OR updated_at > 0)

		  UNION ALL

		  SELECT CASE WHEN regular_refund_last_at > 0
		    THEN regular_refund_last_at + regular_refund_interval_secs
		    ELSE regular_refund_first_at
		  END as next_at
		  FROM vouchers
		  WHERE active = 1 AND regular_refund_first_at > 0
		) sub
		WHERE next_at > ?`,
		now,
	)
	var ts sql.NullInt64
	if err := row.Scan(&ts); err != nil {
		return nil, err
	}
	if !ts.Valid {
		return nil, nil
	}
	t := time.Unix(ts.Int64, 0)
	return &t, nil
}

func (srv *Server) insertRefundTx(dbTx *sql.Tx, voucherID int64, refundCode string, amountMsat, dbTxFee int64) (int64, error) {
	res, err := dbTx.Exec(
		`INSERT INTO refund_txs (voucher_id, refund_code, amount_msat, db_tx_fee, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		voucherID, refundCode, amountMsat, dbTxFee, time.Now().Unix(),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (srv *Server) markVouchersRefunded(dbTx *sql.Tx, ids []int64) error {
	now := time.Now().Unix()
	for _, id := range ids {
		_, err := dbTx.Exec(
			`UPDATE vouchers SET balance_msat = 0, active = 0, updated_at = ? WHERE id = ?`,
			now, id,
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

func deleteVoucherRefundCodes(db dbQuerier, voucherID int64) error {
	_, err := db.Exec(`DELETE FROM voucher_refund_codes WHERE voucher_id = ?`, voucherID)
	return err
}

func (srv *Server) getVoucherIDByPubKey(db dbQuerier, pubKey string) (int64, error) {
	var id int64
	err := db.QueryRow(`SELECT id FROM vouchers WHERE pub_key = ?`, pubKey).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (srv *Server) updateVoucher(dbTx *sql.Tx, id int64, refundAfterSeconds int64, singleUse, transfersOnly bool, maxRedeemMsat int64, uniqueRedemptions, absoluteExpiry bool, regularFirstAt, regularIntervalSecs int64) error {
	_, err := dbTx.Exec(
		`UPDATE vouchers SET
			refund_after_seconds = ?, single_use = ?, transfers_only = ?,
			max_redeem_msat = ?, unique_redemptions = ?, absolute_expiry = ?,
			regular_refund_first_at = ?, regular_refund_interval_secs = ?,
			regular_refund_last_at = 0,
			active = 1, updated_at = ?
		 WHERE id = ?`,
		refundAfterSeconds, boolToInt(singleUse), boolToInt(transfersOnly),
		maxRedeemMsat, boolToInt(uniqueRedemptions), boolToInt(absoluteExpiry),
		regularFirstAt, regularIntervalSecs,
		time.Now().Unix(), id,
	)
	return err
}

func insertVoucherRefundCodes(db dbQuerier, voucherID int64, codes []VoucherRefundCode) error {
	for _, c := range codes {
		if _, err := db.Exec(
			`INSERT INTO voucher_refund_codes (voucher_id, refund_code, share) VALUES (?, ?, ?)`,
			voucherID, c.RefundCode, c.Share,
		); err != nil {
			return err
		}
	}
	return nil
}

func (srv *Server) getRefundCodesForVouchers(ids []int64) (map[int64][]VoucherRefundCode, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	query := `SELECT voucher_id, refund_code, share FROM voucher_refund_codes WHERE voucher_id IN (` +
		strings.Join(placeholders, ",") + `)`
	rows, err := srv.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[int64][]VoucherRefundCode)
	for rows.Next() {
		var voucherID int64
		var rc VoucherRefundCode
		if err := rows.Scan(&voucherID, &rc.RefundCode, &rc.Share); err != nil {
			return nil, err
		}
		out[voucherID] = append(out[voucherID], rc)
	}
	return out, rows.Err()
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
