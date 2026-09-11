## Breez SDK Spark Go upgrade (v0.13.1 → v0.25.0)

### Why this upgrade

`GET /ledger` under-reported the wallet balance and sends intermittently failed with
insufficient funds. Root cause: `GetInfo` reads the SDK's **local cache** in client
mode, and this process never called `SyncWallet` and never handled sync events, so the
cache went stale — and the same cache feeds `SendPayment` leaf selection. SDK releases
between 0.13.1 and 0.25.0 fixed exactly this class of bug (spark-sdk #811 balance lag
on receive, #821 balance flicker during leaf optimization), and v0.25.0 refreshes the
cached balance before emitting `SdkEventPaymentSucceeded`.

### Breaking change (the only one)

- `PrepareSendPaymentRequest.PaymentRequest` changed from `string` to the
  `PaymentRequest` interface. Updated in `handlers.go`:
  `PaymentRequest: spark.PaymentRequestInput{Input: pr}`.

Everything else we use was additive-only (new optional request/response fields left
nil/zero): `SendPaymentRequest`, `ReceivePaymentMethodBolt11Invoice`,
`PrepareLnurlPayRequest/Response`, `LnurlPayRequest`, `ListPaymentsRequest`,
`Payment`, `GetInfoRequest/Response`. The event API shape is unchanged
(`EventListener.OnEvent(SdkEvent)`); `ConnectRequest`, `DefaultConfig`, `Connect`,
`SeedMnemonic`, and `sdkErr()`/`SdkError.AsError()` are unchanged.

### New SDK features adopted

- **Startup sync**: `SyncWallet` runs once after `Connect`/`AddEventListener`, before
  `checkPendingFundTXs` (fatal on error), so catch-up reconciliation and `/ledger`
  start from fresh state. Background sync (client `DefaultConfig`) plus the new event
  handling keep the cache fresh afterwards.
- **Event handling** (`breez.go`): the listener now logs `SdkEventSynced`,
  `SdkEventPaymentPending`, `SdkEventPaymentFailed` (warn, with payment id),
  `SdkEventNewDeposits`, `SdkEventClaimedDeposits`, `SdkEventUnclaimedDeposits`
  (warn, with `ClaimError`), and `SdkEventAutoOptimization` (debug).
- **Payment idempotency keys** (UUIDv5, derived deterministically per tx via the
  hand-rolled `uuid5` helper in `helpers.go` — no new dependency):
  - Redeem: `SendPaymentRequest.IdempotencyKey = uuid5("redeem:{id}")`. A manual retry
    of a redeem with unknown outcome now makes the SDK return the original payment
    instead of risking a double payment.
  - Refund: `LnurlPayRequest.IdempotencyKey = uuid5("refund:{id}")`, stored on
    `refund_txs.idempotency_key` at first attempt and reused on retries. In-flight rows
    **with** a key are auto-requeued at startup (safe); rows **without** a key
    (pre-upgrade) stay manual-review as before.
- **Pending redeem resolution** (`start-up.go`): `redeem_txs.payment_id` stores the SDK
  payment id before the status flip. At startup, pending rows with a payment id are
  resolved via `GetPayment` (completed → confirmed; failed → failed + balance
  restored). Rows without one keep the manual-review warning.
- **`Config.MaxConcurrentClaims`** raised from SDK default 4 to 8 (env
  `MAX_CONCURRENT_CLAIMS`) — recommended for servers with high incoming payment volume.
- **`Disconnect()`** on shutdown (stops SDK background tasks, unregisters listeners).
- **`ListPayments` pagination**: `getPaymentsCompleted` now pages with an explicit
  limit so a busy history window can never truncate pending fund-tx catch-up.
- **Ledger diagnostics**: `GET /ledger` additionally reports `unclaimed_deposits_msat`
  (from `ListUnclaimedDeposits`; omitted if the query fails) and `token_balances`
  (from `GetInfo`; omitted when empty). `wallet_msat` remains `BalanceSats * 1000`
  (spendable Spark leaves only).

### New SDK features deliberately NOT adopted

- **Stable balance / token conversions / `FeePolicy` overrides** — converts sats into
  `TokenBalances`, which would make `BalanceSats` under-report and complicate ledger
  accounting. Default `FeesExcluded` already matches existing behavior.
- **Server mode (`DefaultServerConfig`)** — disables background sync and expects
  webhook-driven, multi-tenant orchestration. For this single long-running instance it
  would make `/ledger` staler, not fresher.
- **Webhooks (`RegisterWebhook`)** — only needed for server mode / multi-instance
  deployments; the in-process event listener covers this service.
- **`PrepareSendBatch`/`SendBatch`** — token-only batches; cannot pay bolt11 invoices.
- **Cross-chain sends/receives, payment links, `BuyBitcoin`, lightning-address
  registration, contacts, fiat rates, client-side `LnurlWithdraw`, `ClaimHtlcPayment`**
  — outside this service's scope.
- **Unilateral-exit APIs** (`ExportUnilateralExitState`, `PrepareUnilateralExit`,
  `UnilateralExit`, …) — disaster recovery only. If ever needed: export exit state
  periodically and store it outside this host; not wired into the service.
- **`LeafOptimizationConfig.Multiplicity`** — left at default 1. Values above 5 exist
  for high-throughput servers; revisit if payment latency becomes an issue. Note that
  the auto-optimizer briefly locks leaves — this is the remaining source of short
  "balance flicker" windows.
- **`PreferSparkOverLightning`** — left false (privacy tradeoff); our counterparty
  invoices are bolt11 anyway.

### Operational notes

- **Back up `STORAGE_DIRECTORY` before the first run** on the new SDK — Spark's local
  tree store may migrate on open.
- **One writer per mnemonic**: do not spend from another wallet using this server's
  mnemonic. 0.14+ tolerates multi-instance much better (real-time sync), but another
  active wallet still moves/locks leaves and makes this server's balance flap.
  Read-only lookups elsewhere are fine.
- `BalanceSats` semantics: spendable Spark leaves only. Unclaimed on-chain deposits
  and token balances are excluded — compare against other wallets accordingly (the
  new `/ledger` fields expose both).

### How to verify

- Build/tests: `go build ./...`, `go vet ./...`, `go test ./...`
- Runtime:
  - Restart logs show `initial wallet sync complete` and `sdk event: wallet synced`.
  - `GET /ledger` `wallet_msat` matches the reference wallet (minus any
    `unclaimed_deposits_msat` / `token_balances`).
  - Fund a voucher, restart before catch-up: no double credit (idempotent confirm).
  - Refund retried with the same idempotency key returns the same payment.
  - Exercise: invoice creation (fund), redeem (`PrepareSendPayment` → `SendPayment`),
    refund worker (`Parse` → `PrepareLnurlPay` → `LnurlPay`).

---

## Previous upgrade: v0.11.0 → v0.13.1

- **API adaptation (required)**: `PrepareLnurlPayRequest` no longer accepts
  `AmountSats`; it expects `Amount` as a `u128` (Go binding: `*big.Int`). Updated in
  `refund.go` via `new(big.Int).SetUint64(amountSats)`.
- **Amounts and fees are `u128`** (`*big.Int` in Go). This repo converts these to
  msats via `.Int64() * 1000` in a few places — fine for voucher-sized amounts, but it
  assumes values fit in `int64`.
