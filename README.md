# satoshi-note

Self-hosted HTTP API for generating cryptographic Lightning Network vouchers.

## Overview

A voucher is a randomly generated secret paired with a SHA-256-derived public key. The API stores vouchers in a local SQLite database and exposes them over HTTP. Vouchers carry a configurable refund window, an optional refund code, and a single-withdrawal flag — making them suitable as bearer tokens for Lightning Network LNURL-withdraw flows or similar off-chain redemption schemes.

## Requirements

- Go 1.24+

## Configuration

Copy `.env.example` to `.env` and edit as needed. All variables are optional; defaults are shown below.

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP listen port |
| `STORAGE_DIRECTORY` | `./.data` | Directory for the SQLite database |
| `DEFAULT_REFUND_AFTER_SECONDS` | `2592000` | Default voucher expiry (30 days) |
| `MAX_REFUND_AFTER_SECONDS` | `31556952` | Maximum allowed expiry (1 year) |
| `MAX_VOUCHERS` | `100` | Max vouchers per batch request |

## Running

```bash
cp .env.example .env
go build -o satoshi-note .
./satoshi-note
```

## API Reference

### `POST /voucher/create`

Create a single voucher.

**Request body** (all fields optional):

```json
{
  "refund_address": "alice@example.com",
  "refund_after_seconds": 86400,
  "single_withdrawal": false
}
```

**Response** `201 Created`:

```json
{
  "id": 1,
  "secret": "a3f1c2d4e5b6a7f8c9d0e1f2a3b4c5d6",
  "pubkey": "3f1c2d4e5b6a7f8c",
  "refund_code": "alice@example.com",
  "refund_after_seconds": 86400,
  "balance_msat": 0,
  "active": true,
  "single_withdrawal": false,
  "last_tx_at": null
}
```

---

### `POST /voucher/create/{amount}`

Create a batch of vouchers. `{amount}` must be a positive integer no greater than `MAX_VOUCHERS`.

**Request body** (all fields optional):

```json
{
  "refund_code": "batch-2026",
  "expires_after_seconds": 604800,
  "single_withdrawal": true
}
```

**Response** `201 Created` — JSON array of voucher objects (same schema as above).

**Example:**

```bash
curl -s -X POST http://localhost:8080/voucher/create/3 \
  -H "Content-Type: application/json" \
  -d '{"single_withdrawal": true}' | jq .
```

## License

GNU General Public License v3.0 — see [LICENSE](LICENSE).
