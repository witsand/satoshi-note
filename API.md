# Satoshi Note — API Documentation

## Overview

Satoshi Note is a Lightning Network voucher system. Vouchers are created via REST API, funded via LNURL-pay (LUD-06), and redeemed via LNURL-withdraw (LUD-11) or transferred to other vouchers. Vouchers can optionally expire and trigger automated refunds to a Lightning address.

All responses are JSON. All amounts are in **millisatoshis (msat)** unless stated otherwise.

---

## Table of Contents

1. [Server Configuration](#server-configuration)
2. [Feature Flags](#feature-flags)
3. [Fee Configuration](#fee-configuration)
4. [Voucher Concepts](#voucher-concepts)
5. [Voucher Field Reference](#voucher-field-reference)
6. [Voucher Creation Constraints](#voucher-creation-constraints)
7. [Endpoints](#endpoints)
   - [POST /create](#post-create)
   - [POST /status](#post-status)
   - [GET /config](#get-config)
   - [GET /ledger](#get-ledger)
   - [GET /transfer/{secret}/{pubKey}](#get-transfersecretpubkey)
   - [GET /f/{pubKey} — LNURL-pay step 1](#get-fpubkey--lnurl-pay-step-1)
   - [GET /fund/{pubKey}/callback — LNURL-pay step 2](#get-fundpubkeycallback--lnurl-pay-step-2)
   - [GET /verify/{key} — LUD-21 Verify](#get-verifykey--lud-21-verify)
   - [GET /w/{secret} — LNURL-withdraw step 1](#get-wsecret--lnurl-withdraw-step-1)
   - [GET /redeem/{secret}/callback — LNURL-withdraw step 2](#get-redeemsecretcallback--lnurl-withdraw-step-2)
8. [Error Responses](#error-responses)
9. [Rate Limiting](#rate-limiting)
10. [LNURL Flows](#lnurl-flows)
11. [Edge Cases & Behaviours](#edge-cases--behaviours)
12. [Fee Calculations](#fee-calculations)

---

## Server Configuration

All settings are loaded from a `.env` file at startup. Missing required variables cause the server to refuse to start.

### Required Settings

| Variable | Type | Description |
|---|---|---|
| `BASE_URL` | string | Public base URL used in all LNURL callbacks (e.g. `https://example.com`). Must not have a trailing slash. |
| `MNEMONIC` | string | BIP-39 seed phrase for the Breez SDK wallet. |
| `BREEZ_API_KEY` | string | API key for the Breez SDK. |
| `PORT` | string | Port the HTTP server listens on (e.g. `8080`). |
| `STORAGE_DIRECTORY` | string | Directory for the SQLite database and Breez SDK state files. |
| `REDEEM_FEE_BPS` | integer | Fee charged on redemptions, in basis points (100 bps = 1%). Applied to the voucher balance before the payout amount is calculated. |
| `MIN_REDEEM_FEE_MSAT` | integer | Floor for the redeem fee in msat. The calculated fee will never be lower than this value. |
| `INTERNAL_FEE_BPS` | integer | Fee charged on internal transfers, in basis points. |
| `MIN_INTERNAL_FEE_MSAT` | integer | Floor for the internal transfer fee in msat. |
| `MIN_FUND_AMOUNT_MSAT` | integer | Minimum amount accepted in a single funding payment (per voucher). |
| `MAX_FUND_AMOUNT_MSAT` | integer | Maximum balance a single voucher may hold. Defines the ceiling for total funded amounts. |
| `MIN_REDEEM_AMOUNT_MSAT` | integer | Minimum net balance required before a voucher can be redeemed. Vouchers below this threshold cannot be withdrawn. |
| `MAX_VOUCHER_EXPIRE_SECONDS` | integer | Hard cap on `refund_after_seconds` at creation time. Values above this are silently clamped down to this cap. |
| `MAX_VOUCHERS_PER_BATCH` | integer | Maximum number of vouchers that can be created in a single `/create` call. |
| `INVOICE_EXPIRY_SECONDS` | integer | Lifetime of BOLT11 invoices generated for funding, in seconds. |

**Validation rule:** `MIN_FUND_AMOUNT_MSAT` must be strictly less than `MAX_FUND_AMOUNT_MSAT`. The server will not start if this is violated.

### Optional Settings

| Variable | Type | Default | Description |
|---|---|---|---|
| `NETWORK` | string | `regtest` | Lightning network. Set to `mainnet` for production. |
| `PAYMENT_COOLDOWN_MS` | integer | `1000` | Milliseconds to hold the payment mutex after each outgoing payment. Prevents rapid consecutive payments. Must be ≥ 0. |
| `REFUND_WORKER_INTERVAL_SECONDS` | integer | `86400` | How often the refund worker checks for expired vouchers, in seconds. |

---

## Feature Flags

These flags independently enable or disable parts of the system. All default to `false` — the server starts with all operations disabled unless explicitly turned on.

| Variable | Default | Effect when `false` |
|---|---|---|
| `CREATE_ACTIVE` | `false` | `POST /create` returns **503 Service Unavailable**. |
| `FUND_ACTIVE` | `false` | `/f/{pubKey}` and `/fund/…` endpoints return a LNURL error (HTTP 200, status `ERROR`). The `/transfer` endpoint also checks this flag for the destination voucher. |
| `REDEEM_ACTIVE` | `false` | `/w/{secret}` and `/redeem/…` endpoints return a LNURL error. The `/transfer` endpoint also checks this flag on the source voucher. |
| `REFUND_ACTIVE` | `false` | The automated refund worker does not run. Expired vouchers are not refunded. |

Feature flags are checked at request time, not at startup, so they can be toggled without restarting the server.

---

## Fee Configuration

Fees affect what a holder can receive from a voucher.

### Redeem Fee

Applied when a voucher is redeemed via LNURL-withdraw.

```
fee = floor((balance_msat * REDEEM_FEE_BPS / 10000 + 1000) / 1000) * 1000
fee = max(fee, MIN_REDEEM_FEE_MSAT)
```

The `+1000` term adds a 1-sat buffer before rounding to a sat boundary.

- The fee is deducted from the voucher balance to arrive at `maxWithdrawable` returned in the LNURL-withdraw response.
- If `max_redeem_msat` is set on the voucher, `maxWithdrawable` is additionally capped at that value.
- The fee is **reserved** in the voucher balance at the time of the LNURL-withdraw step 1. If the actual Lightning routing fee is less than the reserved fee, the surplus stays in the server's accounting (visible in `/ledger`).

### Internal Transfer Fee

Applied when a voucher is transferred to another voucher via `/transfer`.

```
fee = floor((amount_msat * INTERNAL_FEE_BPS / 10000) / 1000) * 1000
fee = max(fee, floor(MIN_INTERNAL_FEE_MSAT / 1000) * 1000)
```

- The gross amount is deducted from the source voucher. The net amount (gross minus fee) is credited to the destination.
- When the destination is a batch, the net amount is divided equally across all vouchers in the batch. Any remainder from integer division (dust) is discarded.

---

## Voucher Concepts

### Pubkey and Secret

Each voucher has a **pubkey** (the identifier used publicly, e.g. in LNURL-pay URLs) and a **secret** (required for redeeming or transferring out). Both are hex strings. The relationship is:

```
pubKey = hex(sha256(secret)[:len(secret)])
```

The server only stores the pubkey. The secret is the bearer credential for withdrawing funds. **Keep the secret private.**

### Batch

When multiple `pub_keys` are passed to `/create`, all vouchers are created in a single database transaction and share a `batch_id`. A batch can be funded collectively via its LNURL-pay URL (using the batch ID instead of a single pubkey). Funding a batch distributes the payment equally across all vouchers in the batch.

### Expiry

A voucher expires when:

```
updated_at + refund_after_seconds <= current_unix_time
```

`updated_at` is set to the time of the **first funding event**. An unfunded voucher (where `updated_at = 0`) never expires via this mechanism.

When a voucher expires:
- It can no longer be funded, redeemed, or transferred.
- If `REFUND_ACTIVE` is `true`, the refund worker will automatically pay the remaining balance to the `refund_code` Lightning address (if any).

### Refund Code

Every voucher stores a `refund_code`, which is a Lightning address (email format) or LNURL. This is used by the automated refund worker when a voucher expires. The refund code is set at creation time and cannot be changed.

---

## Voucher Field Reference

This table covers every voucher-related field, where it comes from, and when it appears in API responses.

| Field | Set at | `/create` response | `/status` response | Notes |
|---|---|---|---|---|
| `pubkey` | Creation | Yes | Key of the status map | The public identifier. |
| `secret` | Creation | Yes | Never | **Bearer credential — keep private.** |
| `batch_id` | Creation | Yes | Never | Shared by all vouchers created in the same call. |
| `refund_code` | Creation | Yes | Never | Lightning address or LNURL for automated refunds. |
| `refund_after_seconds` | Creation | Yes | Never | Capped at `MAX_VOUCHER_EXPIRE_SECONDS`. The expiry TTL, counting from first funding. |
| `single_use` | Creation | Yes | Never | If `true`, min and max withdrawable are both set to the full redeemable balance (fixed-amount withdrawal). |
| `transfers_only` | Creation | Yes | Never | If `true`, LNURL-withdraw is blocked; only internal transfers are allowed. |
| `max_redeem_msat` | Creation | Yes | Never | Per-operation cap (0 = unlimited). Applies to both redeem and transfer operations. |
| `unique_redemptions` | Creation | Yes | Never | If `true`, requires a `fingerprint` on every `/status` and `/transfer` call, and each fingerprint may only redeem once. Implies `transfers_only = true`. |
| `fund_url_prefix` | Derived | Yes | Never | Base URL for LNURL-pay. Append pubkey to get the LNURL-pay URL for a single voucher, or batch_id for the batch. |
| `withdraw_url_prefix` | Derived | Yes | Never | Base URL for LNURL-withdraw. Append the secret to get the claim URL. |
| `claim_lnurl` | Derived | Yes | Never | Bech32-encoded LNURL-withdraw URL, ready to use. |
| `raw_balance_msat` | Funding | Never | Always | The actual stored balance. If `max_redeem_msat > 0`, this is capped at `max_redeem_msat` in the response. |
| `balance_msat` | Computed | Never | Only if `transfers_only = false` | Net redeemable amount after fees. Omitted when the voucher can only be transferred. |
| `expires_at` | Funding | Never | Always | Unix timestamp of expiry (`updated_at + refund_after_seconds`). `0` if the voucher has never been funded (expiry not yet started). |
| `active` | DB state | Never | Always | `false` when the voucher has been deactivated (e.g., fully used up in a single-use scenario). |
| `refunded` | DB state | Never | Always | `true` once the refund worker has successfully paid out the balance. |
| `refund_pending` | DB state | Never | Always | `true` if a refund transaction has been initiated but not yet confirmed. |

### Notes on `balance_msat` vs `raw_balance_msat`

- `raw_balance_msat` is the stored balance, possibly capped at `max_redeem_msat`.
- `balance_msat` is the amount the holder will actually receive after the server's redeem fee is deducted. This field is **only present when `transfers_only = false`**.
- Use `raw_balance_msat` to check if a voucher has been funded. Use `balance_msat` to know what a holder will receive on redemption.

---

## Voucher Creation Constraints

The following rules apply at creation time. Violations return **422 Unprocessable Entity**.

| Combination | Allowed | Reason |
|---|---|---|
| `single_use = true` + `max_redeem_msat > 0` | **No** | `single_use` already fixes the amount; a cap is redundant and ambiguous. |
| `unique_redemptions = true` + `transfers_only = false` | **No** | `unique_redemptions` requires `transfers_only = true` (fingerprint-gated vouchers are designed for transfer flows only). |
| `single_use = true` + `transfers_only = true` | Yes | A voucher that can only be transferred once. |
| `max_redeem_msat > 0` + `transfers_only = true` | Yes | A transfer-only voucher with a per-operation cap. |
| `unique_redemptions = true` + `transfers_only = true` | Yes | The canonical form; each fingerprint may transfer out at most once. |
| `single_use = true` + `unique_redemptions = true` | Yes (but `transfers_only` must also be `true`) | |
| Any combination not listed above | Yes | No other cross-field constraints exist at creation. |

### Summary of field independence

- `single_use` and `transfers_only` are fully independent.
- `max_redeem_msat` is independent of all fields **except**: cannot be combined with `single_use`.
- `unique_redemptions` always requires `transfers_only = true` but is otherwise independent.
- `refund_after_seconds`, `refund_code`, and `pub_keys` are always independent of all other fields.

---

## Endpoints

### POST /create

Creates one or more vouchers. All vouchers in a single request share a `batch_id` and are created atomically (all succeed or all fail).

**Auth:** Requires `Authorization: Bearer <ADMIN_KEY>` header.
**Feature flag:** `CREATE_ACTIVE` must be `true`.

#### Request Body

```json
{
  "pub_keys": ["<hex>", ...],
  "refund_code": "<lightning-address-or-lnurl>",
  "refund_after_seconds": 86400,
  "single_use": false,
  "transfers_only": false,
  "max_redeem_msat": 0,
  "unique_redemptions": false
}
```

| Field | Type | Required | Constraints |
|---|---|---|---|
| `pub_keys` | array of hex strings | Yes | 1 to `MAX_VOUCHERS_PER_BATCH` entries. Each must be a valid hex string of 16–32 bytes (32–64 hex characters). Must be unique within the request. |
| `refund_code` | string | Yes | Email-format Lightning address (e.g. `user@domain.com`) or a raw LNURL string. Stored lowercased. |
| `refund_after_seconds` | integer | Yes | Must be > 0. Silently capped at `MAX_VOUCHER_EXPIRE_SECONDS` if higher. |
| `single_use` | boolean | No | Default `false`. |
| `transfers_only` | boolean | No | Default `false`. |
| `max_redeem_msat` | integer | No | Default `0` (unlimited). Must be ≥ 0. |
| `unique_redemptions` | boolean | No | Default `false`. Requires `transfers_only = true`. |

#### Response — 201 Created

An array with one entry per voucher, in the same order as `pub_keys`.

```json
[
  {
    "pubkey": "aabbcc...",
    "secret": "112233...",
    "batch_id": "ddeeff...",
    "refund_code": "user@example.com",
    "refund_after_seconds": 86400,
    "single_use": false,
    "transfers_only": false,
    "max_redeem_msat": 0,
    "unique_redemptions": false,
    "fund_url_prefix": "https://example.com/f/",
    "withdraw_url_prefix": "https://example.com/w/",
    "claim_lnurl": "LNURL1..."
  }
]
```

- `fund_url_prefix` + `pubkey` = the LNURL-pay URL for a single voucher.
- `fund_url_prefix` + `batch_id` (using the `/fb/` path) = the LNURL-pay URL for the whole batch.
- `withdraw_url_prefix` + `secret` = the LNURL-withdraw URL (same as the decoded `claim_lnurl`).
- `claim_lnurl` is the bech32-encoded version of the withdraw URL. This is what wallets scan.

#### Error Responses

| Status | Reason |
|---|---|
| 400 | `"invalid request body"` — malformed JSON. |
| 400 | `"pub_keys must not be empty"` |
| 400 | `"too many vouchers in batch"` — exceeds `MAX_VOUCHERS_PER_BATCH`. |
| 400 | `"invalid pub_key: must be hex, 16–32 bytes"` — for any individual key. |
| 422 | `"max_redeem_msat cannot be set on a single_use voucher"` |
| 422 | `"unique_redemptions requires transfers_only to be set"` |
| 503 | `CREATE_ACTIVE` is `false`. |
| 500 | Database or internal error. |

---

### POST /status

Returns the current state of one or more vouchers.

**Auth:** None.

#### Request Body

```json
{
  "pubkeys": ["<hex>", ...],
  "fingerprint": "<string>"
}
```

| Field | Type | Required | Constraints |
|---|---|---|---|
| `pubkeys` | array of strings | Yes | Maximum 500 entries. |
| `fingerprint` | string | Conditional | Required for vouchers where `unique_redemptions = true`. See below. |

#### Response — 200 OK

A map from pubkey to status object. Vouchers that are not found or are excluded (see below) are silently omitted from the response.

```json
{
  "<pubkey>": {
    "raw_balance_msat": 100000,
    "balance_msat": 95000,
    "expires_at": 1712000000,
    "active": true,
    "refunded": false,
    "refund_pending": false
  }
}
```

#### Response Fields

| Field | Always present | Notes |
|---|---|---|
| `raw_balance_msat` | Yes | The stored balance. If `max_redeem_msat > 0`, capped at `max_redeem_msat` in this response. |
| `balance_msat` | Only if `transfers_only = false` | Net amount the holder receives after the redeem fee. Omitted for transfer-only vouchers. |
| `expires_at` | Yes | Unix timestamp. `0` means the voucher has never been funded and the expiry clock has not started. |
| `active` | Yes | `false` if the voucher is deactivated. |
| `refunded` | Yes | `true` once the automated refund has been paid successfully. |
| `refund_pending` | Yes | `true` when a refund transaction has been created but not yet confirmed. |

#### Unique-Redemption Vouchers and Fingerprint

When a voucher has `unique_redemptions = true`:

- If `fingerprint` is **not provided**: the voucher is **omitted** from the response entirely.
- If `fingerprint` is provided and **has not been used** on this voucher: the voucher is included with its real balance.
- If `fingerprint` is provided and **has already been used** on this voucher: the voucher is included with `raw_balance_msat = 0`, `balance_msat = 0`, and `active = false`.

This allows a client to use `/status` to determine whether a particular user (identified by fingerprint) can still redeem a given voucher.

#### Error Responses

| Status | Reason |
|---|---|
| 400 | `"invalid json"` or `"too many pubkeys"` (> 500). |
| 500 | Database or internal error. |

---

### GET /config

Returns publicly readable server configuration values.

**Auth:** None.

#### Response — 200 OK

```json
{
  "min_fund_amount_msat": 1000,
  "base_url": "https://example.com"
}
```

| Field | Always present | Notes |
|---|---|---|
| `min_fund_amount_msat` | Yes | The minimum amount per funding payment (per voucher). |
| `base_url` | Yes | The server's public base URL. |

---

### GET /ledger

Returns an accounting summary of the server's Lightning wallet and voucher state. Intended for operators.

**Auth:** None. Rate limited (see [Rate Limiting](#rate-limiting)).

#### Response — 200 OK

```json
{
  "sdk_balance_msat": 5000000,
  "vouchers_balance_msat": 4900000,
  "fund_txs_dust_msat": 200,
  "refund_txs_db_tx_fee": 3000,
  "refund_txs_pending_msat": 50000,
  "redeem_txs_db_tx_fee": 12000,
  "transfer_txs_fee_msat": 5000,
  "transfer_txs_dust_msat": 400,
  "health": 47000,
  "vouchers_avg_hours_to_expiry": 18.5,
  "vouchers_with_balance_count": 12
}
```

| Field | Notes |
|---|---|
| `sdk_balance_msat` | Current balance of the Breez SDK Lightning wallet. |
| `vouchers_balance_msat` | Sum of all voucher balances in the database. |
| `fund_txs_dust_msat` | Cumulative rounding dust from batch funding distributions (sat-level rounding per voucher). |
| `refund_txs_db_tx_fee` | Total fee amounts allocated to pending refund transactions. |
| `refund_txs_pending_msat` | Sum of balances earmarked for pending (not yet confirmed) refunds. |
| `redeem_txs_db_tx_fee` | Total fees reserved from redemptions (the surplus if actual routing fee < reserved fee stays here). |
| `transfer_txs_fee_msat` | Total fees collected from internal transfers. |
| `transfer_txs_dust_msat` | Cumulative rounding dust from batch transfer distributions. |
| `health` | `sdk_balance - vouchers_balance - refund_txs_pending`. A positive number indicates the server holds more than it owes. |
| `vouchers_avg_hours_to_expiry` | Average time until expiry across all active vouchers with balance. |
| `vouchers_with_balance_count` | Number of vouchers that currently hold a positive balance. |

---

### GET /transfer/{secret}/{pubKey}

Transfers funds from one voucher (identified by its secret) to another voucher or batch (identified by pubkey or batch ID).

**Auth:** None. The `secret` is the bearer credential.
**Feature flags:** Both `FUND_ACTIVE` and `REDEEM_ACTIVE` must be `true`.

#### URL Parameters

| Parameter | Required | Description |
|---|---|---|
| `secret` | Yes | The secret of the source voucher. |
| `pubKey` | Yes | The pubkey of the destination voucher, or a batch ID to distribute across a batch. |

#### Query Parameters

| Parameter | Required | Description |
|---|---|---|
| `fingerprint` | Conditional | Required if the source voucher has `unique_redemptions = true`. Must not have been used on this voucher before. |

#### Response — 200 OK

```json
{
  "amount": 100000,
  "fee_msat": 1000,
  "net_msat": 99000
}
```

| Field | Always present | Notes |
|---|---|---|
| `amount` | Yes | Gross amount deducted from the source voucher (msat). |
| `fee_msat` | Yes | Transfer fee charged (msat). |
| `net_msat` | Yes | Amount credited to the destination after fee (msat). For batch destinations, this is divided equally across all vouchers in the batch; dust is discarded. |

#### Constraints

- Source and destination cannot be the same voucher.
- Source cannot be `single_use`.
- Source balance must be > 0.
- If source has `max_redeem_msat > 0`, the transfer amount is capped to that value. Attempting to transfer more returns 422.
- If destination is a batch, the destination must have at least one active, non-expired voucher.
- Both `FUND_ACTIVE` and `REDEEM_ACTIVE` must be `true`; otherwise returns 503.

#### Error Responses

| Status | Reason |
|---|---|
| 400 | `"secret and pubKey are required"`, `"invalid secret"`, `"source and destination cannot be the same voucher"`, `"fingerprint required for this voucher"` |
| 404 | `"source voucher not found"`, `"destination not found"` |
| 409 | `"this fingerprint has already redeemed this voucher"` |
| 422 | `"single-use vouchers cannot transfer funds"`, `"amount exceeds per-transfer limit"` |
| 503 | Transfers disabled (either `FUND_ACTIVE` or `REDEEM_ACTIVE` is `false`). |
| 500 | Database or internal error. |

---

### GET /f/{pubKey} — LNURL-pay step 1

Returns the LNURL-pay parameters for funding a single voucher or batch. This is the URL that a LNURL-pay-capable wallet fetches when it decodes a LNURL.

**Auth:** None.
**Feature flag:** `FUND_ACTIVE` must be `true`; otherwise returns HTTP 200 with an error body.

The `{pubKey}` path parameter may be either:
- A single voucher's pubkey (served at `/f/{pubKey}`)
- A batch ID (served at `/fb/{batchID}`)

#### Response — 200 OK (success)

```json
{
  "tag": "payRequest",
  "callback": "https://example.com/fund/<pubKey>/callback",
  "minSendable": 1000,
  "maxSendable": 100000000,
  "metadata": "[[\"text/plain\",\"Fund a Voucher\"]]"
}
```

| Field | Notes |
|---|---|
| `tag` | Always `"payRequest"`. |
| `callback` | URL the wallet must call with `?amount=<msat>`. |
| `minSendable` | Minimum acceptable payment in msat. For single voucher: `MIN_FUND_AMOUNT_MSAT`. For batch: `MIN_FUND_AMOUNT_MSAT * number_of_vouchers`. |
| `maxSendable` | Maximum acceptable payment in msat. For single voucher: `MAX_FUND_AMOUNT_MSAT - current_balance`. For batch: `min(remaining_capacity_per_voucher) * number_of_vouchers`. |
| `metadata` | LUD-06 metadata string. |

#### Response — 200 OK (error)

```json
{
  "status": "ERROR",
  "reason": "voucher is fully funded"
}
```

| Reason | When |
|---|---|
| `"funding is currently disabled"` | `FUND_ACTIVE` is `false`. |
| `"voucher or batch not found"` | The pubkey or batch ID does not match any active, non-expired record. |
| `"voucher is fully funded"` | The voucher's balance equals `MAX_FUND_AMOUNT_MSAT`. |
| `"batch vouchers are fully funded"` | All vouchers in the batch are at `MAX_FUND_AMOUNT_MSAT`. |

---

### GET /fund/{pubKey}/callback — LNURL-pay step 2

Called by the wallet after presenting the user with the payment request. Creates a BOLT11 invoice the wallet should pay.

**Auth:** None.

#### Query Parameters

| Parameter | Required | Description |
|---|---|---|
| `amount` | Yes | Amount the wallet will pay, in msat. Must fall within the `minSendable`–`maxSendable` range returned in step 1. |

#### Response — 200 OK (success)

```json
{
  "status": "OK",
  "pr": "lnbc...",
  "routes": [],
  "verify": "https://example.com/verify/<key>"
}
```

| Field | Always present | Notes |
|---|---|---|
| `status` | Yes | Always `"OK"` on success. |
| `pr` | Yes | The BOLT11 invoice the wallet should pay. Amount is rounded down to the nearest sat. |
| `routes` | Yes | Always an empty array (no routing hints). |
| `verify` | Yes | LUD-21 verification URL. Poll this to check whether the invoice was paid. |

#### Response — 200 OK (error)

```json
{
  "status": "ERROR",
  "reason": "amount would exceed maximum voucher balance"
}
```

| Reason | When |
|---|---|
| `"funding is currently disabled"` | `FUND_ACTIVE` is `false`. |
| `"invalid amount"` | `amount` query param is missing or not a valid integer. |
| `"amount would exceed maximum voucher balance"` | The requested amount would push the balance above `MAX_FUND_AMOUNT_MSAT`. |
| `"voucher or batch not found"` | No active, non-expired record found for the pubkey or batch. |
| `"failed to create invoice"` | Breez SDK error during invoice creation. |
| `"failed to write fund tx"` | Database write error. |

---

### GET /verify/{key} — LUD-21 Verify

Checks whether a funding invoice has been paid. The `key` is returned in the `verify` field of the LNURL-pay step 2 response.

**Auth:** None.

#### Response — 200 OK (success)

```json
{
  "status": "OK",
  "settled": true,
  "preimage": "aabbcc...",
  "pr": "lnbc..."
}
```

| Field | Always present | Notes |
|---|---|---|
| `status` | Yes | `"OK"` if the key was found. |
| `settled` | Yes | `true` if the payment has been confirmed. `false` if pending. |
| `preimage` | Yes | Payment preimage. Empty string (`""`) if not yet settled. |
| `pr` | Yes | The original BOLT11 invoice. |

#### Response — 200 OK (error)

```json
{
  "status": "ERROR",
  "reason": "not found"
}
```

| Reason | When |
|---|---|
| `"not found"` | No fund transaction record exists for this key. |

---

### GET /w/{secret} — LNURL-withdraw step 1

Returns the LNURL-withdraw parameters for redeeming a voucher. This is the URL that a LNURL-withdraw-capable wallet fetches when it decodes the `claim_lnurl`.

**Auth:** None. The `secret` is the bearer credential.
**Feature flag:** `REDEEM_ACTIVE` must be `true`; otherwise returns HTTP 200 with an error body.

#### Response — 200 OK (success)

```json
{
  "tag": "withdrawRequest",
  "callback": "https://example.com/redeem/<secret>/callback",
  "k1": "aabbcc...",
  "minWithdrawable": 95000,
  "maxWithdrawable": 95000,
  "defaultDescription": "Redeem Voucher"
}
```

| Field | Notes |
|---|---|
| `tag` | Always `"withdrawRequest"`. |
| `callback` | URL the wallet must call with `?k1=<k1>&pr=<bolt11>`. |
| `k1` | Session token. Valid for 30 minutes. Must be passed unmodified in the callback. |
| `minWithdrawable` | Minimum amount the wallet may request, in msat. |
| `maxWithdrawable` | Maximum amount the wallet may request, in msat. |
| `defaultDescription` | Always `"Redeem Voucher"`. |

**Amount calculation:**

```
redeemable = balance - redeem_fee
if max_redeem_msat > 0:
    redeemable = min(redeemable, max_redeem_msat)

maxWithdrawable = redeemable
minWithdrawable = MAX(MIN_REDEEM_AMOUNT_MSAT, redeemable)

if single_use:
    minWithdrawable = maxWithdrawable   # force exact amount
```

For `single_use` vouchers, `minWithdrawable` equals `maxWithdrawable`. The wallet must request exactly that amount.

#### Response — 200 OK (error)

```json
{
  "status": "ERROR",
  "reason": "voucher not found"
}
```

| Reason | When |
|---|---|
| `"redeem is currently disabled"` | `REDEEM_ACTIVE` is `false`. |
| `"invalid secret"` | The secret is not valid hex. |
| `"voucher not found"` | No active, non-expired voucher found for this secret. |
| `"this voucher can only be transferred, not redeemed"` | Voucher has `transfers_only = true`. |
| `"voucher balance too low"` | Balance minus fee is below `MIN_REDEEM_AMOUNT_MSAT`. |

---

### GET /redeem/{secret}/callback — LNURL-withdraw step 2

Called by the wallet to actually pay out the voucher balance to a BOLT11 invoice the wallet provides.

**Auth:** None. The `secret` and `k1` together authenticate the session.
**Feature flag:** `REDEEM_ACTIVE` must be `true`.

#### Query Parameters

| Parameter | Required | Description |
|---|---|---|
| `k1` | Yes | The session token returned in step 1. Valid for 30 minutes. |
| `pr` | Yes | A BOLT11 invoice the wallet generated. Must specify an amount. Zero-amount invoices are not supported. |

#### Response — 200 OK (success)

```json
{
  "status": "OK"
}
```

#### Payment Processing

When the callback is received:

1. The server acquires the payment semaphore. If the server is busy with another payment, waits up to 5 seconds. Returns 503 if timeout.
2. Validates the `k1`: must exist in the database, not have been used before, and be within the 30-minute window.
3. Parses the `pr` and reads the amount. Zero-amount invoices are rejected.
4. Validates amount: `pr_amount + redeem_fee ≤ voucher_balance`. If not, returns error.
5. If `max_redeem_msat > 0`, validates `pr_amount ≤ max_redeem_msat`.
6. Checks that the Lightning routing fee estimate from the SDK (`PrepareSendPayment`) does not exceed the server's reserved fee.
7. Deducts `pr_amount + redeem_fee` from the voucher balance in the database.
8. Attempts the Lightning payment. If payment fails, restores the full deducted amount.
9. On success, records the transaction with actual fees paid.

The semaphore is released after `PAYMENT_COOLDOWN_MS` milliseconds.

#### Response — 200 OK (error)

```json
{
  "status": "ERROR",
  "reason": "payment failed"
}
```

| Reason | When |
|---|---|
| `"redeem is currently disabled"` | `REDEEM_ACTIVE` is `false`. |
| `"invalid secret"` | Secret is not valid hex. |
| `"missing k1"` or `"missing pr"` | Query params absent. |
| `"invalid or expired k1"` | k1 not found, already used, or older than 30 minutes. |
| `"zero-amount invoices are not supported"` | The wallet sent an invoice without a fixed amount. |
| `"voucher not found"` | No active, non-expired voucher for this secret. |
| `"this voucher can only be transferred, not redeemed"` | `transfers_only = true`. |
| `"redeem amount exceeds per-redeem limit"` | Amount exceeds `max_redeem_msat`. |
| `"routing fee too high"` | SDK fee estimate exceeds the reserved redeem fee. |
| `"redeem amount exceeds voucher balance after fees"` | Amount + fee > current balance. |
| `"payment failed"` | Lightning payment attempt failed at the SDK level. |
| `"internal db error"` | Database error during transaction. |

#### Response — 503 Service Unavailable

```json
{
  "status": "ERROR",
  "reason": "server busy, please retry"
}
```

Returned when the payment semaphore cannot be acquired within 5 seconds. The wallet should retry after a short delay.

---

## Error Responses

### LNURL Endpoints

LNURL endpoints (`/f/`, `/fb/`, `/fund/`, `/verify/`, `/w/`, `/redeem/`) **always return HTTP 200**, even on error. Errors are indicated by the response body:

```json
{
  "status": "ERROR",
  "reason": "human-readable message"
}
```

This is required by the LNURL specification. Wallets check `status` in the body, not the HTTP status code.

**Exception:** The `/redeem/` callback returns **HTTP 503** (not 200) when the payment semaphore times out.

### Non-LNURL Endpoints

All other endpoints (`/create`, `/status`, `/transfer`, `/config`, `/ledger`) use standard HTTP status codes. The error body format is the same:

```json
{
  "status": "ERROR",
  "reason": "human-readable message"
}
```

| Code | Meaning |
|---|---|
| 400 | Bad request (invalid JSON, invalid parameter values). |
| 409 | Conflict (e.g. fingerprint already used). |
| 422 | Constraint violation (e.g. incompatible voucher flags). |
| 503 | Feature disabled by server flag. |
| 500 | Internal server or database error. |

---

## Rate Limiting

Rate limits are applied per client IP address using a token-bucket algorithm. Buckets are automatically cleaned up after 10 minutes of inactivity.

**Client IP resolution** (in priority order):
1. `CF-Connecting-IP` header (when behind Cloudflare)
2. `X-Real-IP` header (when behind nginx/Caddy)
3. `RemoteAddr` (direct connections)

| Endpoint group | Rate | Burst |
|---|---|---|
| `/create`, `/ledger` | 1 request / 60 seconds | 2 |
| `/status`, `/config`, `/transfer` | 2 requests / second | 10 |
| `/f/`, `/fb/`, `/fund/`, `/verify/`, `/w/`, `/redeem/` | 5 requests / second | 20 |

Exceeded limits return **HTTP 429 Too Many Requests**.

---

## LNURL Flows

### Funding a Voucher (LUD-06 Pay)

```
Wallet                         Server
  |                               |
  |  GET /f/{pubKey}              |
  |------------------------------>|
  |  { tag, callback, minSendable, maxSendable, metadata }
  |<------------------------------|
  |                               |
  |  (user confirms amount)       |
  |                               |
  |  GET /fund/{pubKey}/callback?amount=<msat>
  |------------------------------>|
  |  { status:"OK", pr, routes, verify }
  |<------------------------------|
  |                               |
  |  (wallet pays invoice)        |
  |                               |
  |  GET /verify/{key}  (polling) |
  |------------------------------>|
  |  { status:"OK", settled:true, preimage, pr }
  |<------------------------------|
```

**Acceptable amounts:** The wallet must send an amount in the range `[minSendable, maxSendable]` as returned in step 1. The actual invoice is created for the requested amount rounded down to the nearest satoshi.

**Batch funding:** When funding a batch via `/fb/{batchID}`, the amount is divided equally among all vouchers. Integer rounding means each voucher receives `floor(amount / count)` msat (rounded down to the nearest sat). Any dust remainder is recorded but not credited to vouchers.

### Redeeming a Voucher (LUD-11 Withdraw)

```
Wallet                         Server
  |                               |
  |  GET /w/{secret}              |
  |------------------------------>|
  |  { tag, callback, k1, minWithdrawable, maxWithdrawable, defaultDescription }
  |<------------------------------|
  |                               |
  |  (wallet generates invoice)   |
  |                               |
  |  GET /redeem/{secret}/callback?k1=<k1>&pr=<bolt11>
  |------------------------------>|
  |  { status:"OK" }              |
  |<------------------------------|
```

**k1 validity:** The session token `k1` is valid for **30 minutes** from the time `/w/{secret}` was called. If the wallet does not call the callback within this window, the session expires and the wallet must restart the flow.

**Amount:** For `single_use` vouchers, the wallet must use exactly `minWithdrawable = maxWithdrawable`. For regular vouchers, any amount between `minWithdrawable` and `maxWithdrawable` is accepted.

---

## Edge Cases & Behaviours

### Funded vs. Unfunded Vouchers

A voucher starts with `balance = 0` and `updated_at = 0`. The expiry clock (`refund_after_seconds`) does not start until the first funding event. An unfunded voucher has `expires_at = 0` in `/status`.

An unfunded voucher cannot be redeemed (balance is below `MIN_REDEEM_AMOUNT_MSAT`). It can still appear in `/status` with `active = true`.

### Expired Vouchers

A voucher expires when the wall-clock time exceeds `updated_at + refund_after_seconds`. Once expired:

- It is invisible to LNURL-pay (`/f/`, `/fund/`): returns `"voucher or batch not found"`.
- It is invisible to LNURL-withdraw (`/w/`, `/redeem/`): returns `"voucher not found"`.
- It still appears in `/status` with its last known balance and `active = false`.
- The automated refund worker (if `REFUND_ACTIVE = true`) will pay any remaining balance to `refund_code`.

### Refund Behaviour

The refund worker runs on the interval set by `REFUND_WORKER_INTERVAL_SECONDS`. For each expired voucher with `balance > 0` and `refunded = false`:

1. Groups vouchers by `refund_code` and sums their balances.
2. Deducts a `dbTxFee` (same calculation as the redeem fee) to cover the Lightning payment.
3. If the net amount is below `MIN_REDEEM_AMOUNT_MSAT`, the refund is **skipped** for that voucher in this run.
4. Creates a `refund_tx` record and zeroes out the voucher balance, all in one database transaction.
5. Attempts to pay the `refund_code` via Lightning. If the payment fails, the error is recorded but the voucher's balance remains zeroed (i.e., funds are not restored).

`refund_pending = true` in `/status` indicates step 4 completed but step 5 has not confirmed. `refunded = true` indicates the payment was confirmed.

### Duplicate Fingerprints (unique_redemptions)

When `unique_redemptions = true` on a voucher, every `/transfer` call must include a `fingerprint` query parameter. The server records the (voucher, fingerprint) pair in the database.

- If the same fingerprint is submitted for the same voucher a second time: **HTTP 409**.
- Race conditions are handled atomically: the insert uses `INSERT OR IGNORE`, and if zero rows are affected, the request is treated as a duplicate.
- The fingerprint is scoped to a single voucher. The same fingerprint can be used across different vouchers.

### Single-Use Vouchers

When `single_use = true`:

- `minWithdrawable = maxWithdrawable` in the LNURL-withdraw response. The wallet must withdraw exactly the full redeemable amount.
- Cannot have `max_redeem_msat > 0` set (rejected at creation).
- Cannot be a transfer source (`/transfer` returns 422).
- After a successful withdrawal the voucher may be set inactive depending on resulting balance.

### Startup Reconciliation

When the server starts, it runs `checkPendingFundTXs()`:

1. Finds all fund transactions in the database with `status = "pending"`.
2. Queries the Breez SDK for all completed receive payments since 10 minutes before the oldest pending transaction.
3. Matches payments by invoice (the `pr` field).
4. For any matched payments, updates the fund transaction status and credits the voucher balance.

This ensures that vouchers funded while the server was offline (e.g., after a restart) are correctly credited.

### Concurrent Payment Protection

The server uses a single payment semaphore to ensure only one outgoing Lightning payment is in flight at a time.

- `/redeem` callback: acquires semaphore with a 5-second timeout. Returns 503 if the semaphore cannot be acquired.
- Refund worker: acquires semaphore without timeout (will wait indefinitely).
- After a payment completes, the semaphore is held for an additional `PAYMENT_COOLDOWN_MS` milliseconds before releasing.

---

## Fee Calculations

### Redeem Fee

Used when a voucher is redeemed via LNURL-withdraw or when calculating the net refund amount.

```
step1: fee = (balance_msat * REDEEM_FEE_BPS / 10000) + 1000
step2: fee = floor(fee / 1000) * 1000          # round down to nearest sat
step3: fee = max(fee, MIN_REDEEM_FEE_MSAT)     # apply floor
```

The fee is subtracted from the voucher balance to determine `maxWithdrawable`. The server reserves this fee at payment time. If the actual Lightning routing fee is lower, the surplus accrues to the server (visible as `redeem_txs_db_tx_fee` in `/ledger`).

### Internal Transfer Fee

Used when transferring between vouchers via `/transfer`.

```
step1: fee = amount_msat * INTERNAL_FEE_BPS / 10000
step2: fee = floor(fee / 1000) * 1000                   # round down to nearest sat
step3: fee = max(fee, floor(MIN_INTERNAL_FEE_MSAT / 1000) * 1000)  # apply floor
```

The gross amount is taken from the source. The destination receives `amount - fee`. For batch destinations, the net is split equally per voucher with sat-level rounding; any dust remainder is discarded and tracked in `/ledger`.

### Example: Voucher with 100,000 msat balance

Assume `REDEEM_FEE_BPS = 100` (1%) and `MIN_REDEEM_FEE_MSAT = 2000`.

```
step1: fee = (100000 * 100 / 10000) + 1000 = 1000 + 1000 = 2000
step2: fee = floor(2000 / 1000) * 1000 = 2000
step3: fee = max(2000, 2000) = 2000

maxWithdrawable = 100000 - 2000 = 98000 msat
```

`raw_balance_msat` in `/status` = 100000
`balance_msat` in `/status` = 98000
