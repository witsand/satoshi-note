## Breez SDK Spark Go upgrade (v0.11.0 → v0.13.1)

### What changed in this repo
- **Dependency bump**: `github.com/breez/breez-sdk-spark-go` upgraded to **`v0.13.1`** in `go.mod`.
- **API adaptation (required)**: `PrepareLnurlPayRequest` no longer accepts `AmountSats`; it now expects `Amount` as a `u128` (Go binding: `*big.Int`).
  - Updated in `refund.go` to convert sats → `*big.Int` via `new(big.Int).SetUint64(amountSats)`.

### Notes / risks to keep in mind
- **Amounts and fees are `u128`** in the updated bindings (represented as `*big.Int` in Go). This repo still converts these to msats via `.Int64() * 1000` in a few places. That’s fine for typical voucher-sized amounts, but it assumes values fit in `int64`.
- The existing `sdkErr()` helper remains valid with the regenerated UniFFI bindings.

### How to verify
- Build/tests:
  - `go test ./...`
  - `go test -race ./...`
  - `go build ./...`
- Runtime sanity (regtest/dev recommended):
  - Start the service and exercise: invoice creation (fund), payment succeeded event reconciliation, redeem send (`PrepareSendPayment` → `SendPayment`), refund worker LNURL-pay (`Parse` → `PrepareLnurlPay` → `LnurlPay`).

