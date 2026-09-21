package main

import (
	"net/http"
)

// handleAdminUI serves the single-page admin console. The page is static and
// unauthenticated; it asks for the admin token in the browser and sends it as a
// bearer token to the /admin/* APIs, which enforce ADMIN_SECRET.
func handleAdminUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(adminUIHTML))
}

const adminUIHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Satoshi Note — Admin</title>
<style>
  :root {
    --bg: #0f1115; --panel: #171a21; --panel2: #1e222b; --border: #2a2f3a;
    --text: #e6e9ef; --muted: #9aa3b2; --accent: #f7931a; --ok: #2fbf71;
    --bad: #e5484d; --mono: ui-monospace, SFMono-Regular, Menlo, monospace;
  }
  * { box-sizing: border-box; }
  body { margin: 0; background: var(--bg); color: var(--text);
    font: 15px/1.5 system-ui, -apple-system, Segoe UI, Roboto, sans-serif; }
  header { padding: 16px 20px; border-bottom: 1px solid var(--border);
    display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }
  header h1 { font-size: 17px; margin: 0; font-weight: 600; }
  header .dot { width: 9px; height: 9px; border-radius: 50%; background: var(--muted); }
  header .dot.ok { background: var(--ok); } header .dot.bad { background: var(--bad); }
  main { max-width: 900px; margin: 0 auto; padding: 20px; }
  #tokenGate { background: var(--panel); border: 1px solid var(--border);
    border-radius: 12px; padding: 24px; max-width: 420px; margin: 60px auto; }
  #tokenGate h2 { margin: 0 0 6px; font-size: 16px; }
  #tokenGate p { color: var(--muted); margin: 0 0 16px; font-size: 14px; }
  input, textarea { width: 100%; background: var(--panel2); border: 1px solid var(--border);
    color: var(--text); border-radius: 8px; padding: 10px 12px; font: inherit; }
  input:focus, textarea:focus { outline: 1px solid var(--accent); border-color: var(--accent); }
  button { background: var(--accent); color: #1a1205; border: 0; border-radius: 8px;
    padding: 10px 16px; font: inherit; font-weight: 600; cursor: pointer; }
  button.ghost { background: var(--panel2); color: var(--text); border: 1px solid var(--border); }
  button:disabled { opacity: .5; cursor: default; }
  .tabs { display: flex; gap: 6px; margin: 18px 0; }
  .tabs button { background: var(--panel2); color: var(--muted); border: 1px solid var(--border); }
  .tabs button.active { background: var(--accent); color: #1a1205; border-color: var(--accent); }
  .card { background: var(--panel); border: 1px solid var(--border); border-radius: 12px;
    padding: 16px; margin-bottom: 14px; }
  .stat { display: flex; justify-content: space-between; gap: 12px; padding: 6px 0;
    border-bottom: 1px dashed var(--border); }
  .stat:last-child { border-bottom: 0; }
  .stat .k { color: var(--muted); }
  .stat .v { font-family: var(--mono); }
  details { background: var(--panel); border: 1px solid var(--border); border-radius: 12px;
    margin-bottom: 10px; overflow: hidden; }
  details summary { cursor: pointer; padding: 13px 16px; font-weight: 600; list-style: none;
    display: flex; justify-content: space-between; align-items: center; }
  details summary::-webkit-details-marker { display: none; }
  details summary::after { content: "+"; color: var(--muted); font-weight: 400; }
  details[open] summary::after { content: "–"; }
  details .body { padding: 0 16px 14px; }
  .grid { display: grid; grid-template-columns: 1fr 1fr; gap: 6px 24px; }
  @media (max-width: 560px) { .grid { grid-template-columns: 1fr; } }
  .badge { font-size: 12px; padding: 2px 9px; border-radius: 99px; font-weight: 600; }
  .badge.ok { background: rgba(47,191,113,.15); color: var(--ok); }
  .badge.bad { background: rgba(229,72,77,.15); color: var(--bad); }
  label { display: block; font-size: 13px; color: var(--muted); margin: 14px 0 6px; }
  .row { display: flex; gap: 10px; flex-wrap: wrap; margin-top: 16px; }
  .msg { margin-top: 14px; font-size: 14px; white-space: pre-wrap; word-break: break-word; }
  .msg.err { color: var(--bad); } .msg.ok { color: var(--ok); }
  .mono { font-family: var(--mono); font-size: 13px; }
  .inv { background: var(--panel2); border: 1px solid var(--border); border-radius: 8px;
    padding: 10px 12px; margin-top: 12px; word-break: break-all; font-family: var(--mono); font-size: 12px; }
  .hidden { display: none; }
  .sub { color: var(--muted); font-size: 13px; font-weight: 400; margin-left: 8px; }
</style>
</head>
<body>
<header>
  <span class="dot" id="statusDot"></span>
  <h1>Satoshi Note — Admin</h1>
  <span class="badge hidden" id="balancedBadge"></span>
  <span style="flex:1"></span>
  <button class="ghost hidden" id="signOut">Sign out</button>
</header>

<div id="tokenGate">
  <h2>Admin token</h2>
  <p>Enter the server <span class="mono">ADMIN_SECRET</span> to continue.</p>
  <input type="password" id="tokenInput" placeholder="admin token" autocomplete="off">
  <div class="row">
    <button id="tokenGo">Continue</button>
  </div>
  <div class="msg err" id="tokenErr"></div>
</div>

<main id="app" class="hidden">
  <div class="tabs">
    <button data-tab="ledger" class="active">Ledger</button>
    <button data-tab="funds">Deposit / Withdraw</button>
  </div>

  <section id="tab-ledger">
    <div class="row" style="margin:0 0 14px">
      <button class="ghost" id="refreshLedger">Refresh</button>
    </div>
    <div id="ledgerSummary"></div>
    <div id="ledgerTopics"></div>
  </section>

  <section id="tab-funds" class="hidden">
    <div class="card">
      <h3 style="margin:0 0 4px">Deposit</h3>
      <div class="sub">Create an invoice to top up the wallet. Pay it from a <b>different</b> wallet.</div>
      <label>Amount (msat) — leave blank to use the current deficit</label>
      <input id="depAmount" inputmode="numeric" placeholder="e.g. 50000">
      <div class="row"><button id="depGo">Create deposit invoice</button></div>
      <div id="depOut"></div>
    </div>

    <div class="card">
      <h3 style="margin:0 0 4px">Withdraw equity</h3>
      <div class="sub">Pay out accumulated fees + dust. Capped so voucher/refund holds stay covered.</div>
      <label>Destination — bolt11 invoice, LNURL-pay, or lightning address</label>
      <input id="wdDest" placeholder="lnbc… / user@domain / lnurl…">
      <label>Amount (msat) — leave blank to withdraw the full available amount</label>
      <input id="wdAmount" inputmode="numeric" placeholder="e.g. 50000">
      <div class="row"><button id="wdGo">Withdraw</button></div>
      <div id="wdOut"></div>
    </div>
  </section>
</main>

<script>
const $ = (id) => document.getElementById(id);
let token = sessionStorage.getItem("adminToken") || "";

function fmtMsat(v) {
  if (v === null || v === undefined) return "—";
  const n = Number(v);
  const sats = n / 1000;
  const s = sats.toLocaleString(undefined, { maximumFractionDigits: 3 });
  return s + " sat";
}
function fmtSecs(v) {
  if (v === null || v === undefined) return "—";
  const n = Number(v);
  const sign = n < 0 ? "-" : "";
  const a = Math.abs(n);
  if (a >= 86400) return sign + (a / 86400).toFixed(1) + " d";
  if (a >= 3600) return sign + (a / 3600).toFixed(1) + " h";
  if (a >= 60) return sign + (a / 60).toFixed(1) + " m";
  return sign + Math.round(a) + " s";
}
function esc(s) { return String(s).replace(/[&<>"]/g, c => ({"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;"}[c])); }

async function api(path, opts = {}) {
  const res = await fetch(path, {
    ...opts,
    headers: { "Authorization": "Bearer " + token, "Content-Type": "application/json", ...(opts.headers || {}) },
  });
  let body = null;
  try { body = await res.json(); } catch (e) {}
  if (!res.ok) {
    const reason = body && (body.reason || body.error) ? (body.reason || body.error) : ("HTTP " + res.status);
    const err = new Error(reason); err.status = res.status; throw err;
  }
  return body;
}

function stat(k, v) { return '<div class="stat"><span class="k">' + k + '</span><span class="v">' + v + "</span></div>"; }
function topic(title, inner, open) {
  return '<details' + (open ? " open" : "") + "><summary>" + title + "</summary><div class=\"body\">" + inner + "</div></details>";
}
function bucket(b, label) {
  if (!b) return "";
  return stat(label + " count", b.count) + stat(label, fmtMsat(b.amount_msat)) + stat(label + " fee reserved", fmtMsat(b.fee_reserved_msat));
}

function renderLedger(s) {
  const dot = $("statusDot"), badge = $("balancedBadge");
  dot.className = "dot " + (s.balanced ? "ok" : "bad");
  badge.className = "badge " + (s.balanced ? "ok" : "bad");
  badge.textContent = s.balanced ? "balanced" : "deficit";

  $("ledgerSummary").innerHTML =
    '<div class="card">' +
    stat("Wallet", fmtMsat(s.wallet_msat)) +
    stat("Explained by books", fmtMsat(s.explained_msat)) +
    stat("Imbalance (wallet − books)", fmtMsat(s.imbalance_msat)) +
    (s.unclaimed_deposits_msat != null ? stat("Unclaimed on-chain deposits", fmtMsat(s.unclaimed_deposits_msat)) : "") +
    "</div>";

  const L = s.liabilities || {}, E = s.equity || {}, A = s.activity || {}, V = s.vouchers || {};
  const topics = [];

  topics.push(topic("Vouchers <span class='sub'>" + (V.with_balance || 0) + " with balance</span>",
    '<div class="grid">' +
    stat("Total", V.total) + stat("Active", V.active) +
    stat("Inactive", V.inactive) + stat("Expired w/ balance", V.expired_with_balance) +
    stat("Active balance", fmtMsat(L.vouchers && L.vouchers.active && L.vouchers.active.balance_msat)) +
    stat("Inactive balance", fmtMsat(L.vouchers && L.vouchers.inactive && L.vouchers.inactive.balance_msat)) +
    stat("Avg time to expiry", fmtSecs(L.vouchers && L.vouchers.active && L.vouchers.active.avg_secs_to_expiry)) +
    "</div>", true));

  const rf = L.refunds || {};
  let refundInner = '<div class="grid">' +
    bucket(rf.pending, "Pending") + bucket(rf.in_flight, "In flight") + bucket(rf.abandoned, "Abandoned") +
    stat("Held (refunds)", fmtMsat(rf.held_msat)) + "</div>";
  if (rf.abandoned && rf.abandoned.items && rf.abandoned.items.length) {
    refundInner += "<div class='mono' style='margin-top:8px'>" +
      rf.abandoned.items.map(i => esc(i.refund_code) + " — " + fmtMsat(i.amount_msat) + (i.error_msg ? " — " + esc(i.error_msg) : "")).join("<br>") + "</div>";
  }
  topics.push(topic("Refunds <span class='sub'>held " + fmtMsat(rf.held_msat) + "</span>", refundInner));

  const rd = L.redeems_pending || {};
  topics.push(topic("Pending redeems <span class='sub'>" + (rd.count || 0) + "</span>",
    '<div class="grid">' + stat("Count", rd.count) + stat("Amount", fmtMsat(rd.amount_msat)) +
    stat("Fee reserved", fmtMsat(rd.fee_reserved_msat)) + stat("Held", fmtMsat(rd.held_msat)) + "</div>"));

  topics.push(topic("Equity <span class='sub'>available " + fmtMsat(E.available_msat) + "</span>",
    '<div class="grid">' +
    stat("Available to withdraw", fmtMsat(E.available_msat)) +
    stat("Withdrawn (all time)", fmtMsat(E.withdrawn_msat)) +
    stat("Deposits (all time)", fmtMsat(E.deposits_msat)) +
    stat("Fees total", fmtMsat(E.fees && E.fees.total_msat)) +
    stat("· transfer", fmtMsat(E.fees && E.fees.transfer_msat)) +
    stat("· redeem net", fmtMsat(E.fees && E.fees.redeem_net_msat)) +
    stat("· refund net", fmtMsat(E.fees && E.fees.refund_net_msat)) +
    stat("Dust total", fmtMsat(E.dust && E.dust.total_msat)) +
    "</div>", true));

  const f = A.fund_txs || {}, rt = A.redeem_txs || {}, tt = A.transfer_txs || {}, op = A.operator_txs || {};
  topics.push(topic("Activity",
    '<div class="grid">' +
    stat("Fund confirmed", (f.confirmed && f.confirmed.count) || 0) +
    stat("Fund confirmed amount", fmtMsat(f.confirmed && f.confirmed.msat)) +
    stat("Fund receive fees (cost, not equity)", fmtMsat(f.confirmed && f.confirmed.fee_msat)) +
    stat("Fund pending", (f.pending && f.pending.count) || 0) +
    stat("Redeem confirmed", (rt.confirmed && rt.confirmed.count) || 0) +
    stat("Redeem failed", (rt.failed && rt.failed.count) || 0) +
    stat("Transfers", tt.count || 0) +
    stat("Pending deposits", (op.pending_deposits && op.pending_deposits.count) || 0) +
    stat("Pending withdraws", (op.pending_withdraws && op.pending_withdraws.count) || 0) +
    "</div>"));

  if (s.token_balances && Object.keys(s.token_balances).length) {
    topics.push(topic("Token balances",
      "<div class='mono'>" + Object.entries(s.token_balances).map(([k, v]) => esc(k) + ": " + esc(v)).join("<br>") + "</div>"));
  }

  $("ledgerTopics").innerHTML = topics.join("");
}

async function loadLedger() {
  try {
    const s = await api("/admin/ledger");
    renderLedger(s);
  } catch (e) {
    if (e.status === 401) { signOut(); return; }
    $("ledgerTopics").innerHTML = '<div class="card msg err">' + esc(e.message) + "</div>";
  }
}

function out(el, msg, isErr) {
  el.className = "msg " + (isErr ? "err" : "ok");
  el.textContent = msg;
}

async function doDeposit() {
  const el = $("depOut"); el.innerHTML = "";
  const amt = $("depAmount").value.trim();
  $("depGo").disabled = true;
  try {
    const body = amt ? { amount_msat: Number(amt) } : {};
    const r = await api("/admin/deposit", { method: "POST", body: JSON.stringify(body) });
    el.innerHTML = '<div class="msg ok">Invoice created — pay it from a different wallet.</div>' +
      '<div class="inv">' + esc(r.pr) + "</div>" +
      '<div class="msg">Amount: ' + fmtMsat(r.amount_msat) + " · deficit: " + fmtMsat(r.deficit_msat) +
      (r.receive_fee_estimate_msat ? " · receive fee est: " + fmtMsat(r.receive_fee_estimate_msat) : "") + "</div>";
    loadLedger();
  } catch (e) { out(el, e.message, true); }
  $("depGo").disabled = false;
}

async function doWithdraw() {
  const el = $("wdOut"); el.innerHTML = "";
  const dest = $("wdDest").value.trim();
  const amt = $("wdAmount").value.trim();
  if (!dest) { out(el, "destination is required", true); return; }
  $("wdGo").disabled = true;
  try {
    const body = { destination: dest };
    if (amt) body.amount_msat = Number(amt);
    const r = await api("/admin/withdraw", { method: "POST", body: JSON.stringify(body) });
    out(el, "Withdraw sent — " + fmtMsat(r.amount_msat), false);
    loadLedger();
  } catch (e) { out(el, e.message, true); }
  $("wdGo").disabled = false;
}

function showApp() {
  $("tokenGate").classList.add("hidden");
  $("app").classList.remove("hidden");
  $("signOut").classList.remove("hidden");
  $("balancedBadge").classList.remove("hidden");
  loadLedger();
}
function signOut() {
  token = ""; sessionStorage.removeItem("adminToken");
  $("app").classList.add("hidden");
  $("signOut").classList.add("hidden");
  $("balancedBadge").classList.add("hidden");
  $("tokenGate").classList.remove("hidden");
  $("statusDot").className = "dot";
}

async function tryToken(t) {
  token = t;
  try { await api("/admin/ledger"); sessionStorage.setItem("adminToken", t); showApp(); }
  catch (e) { $("tokenErr").textContent = e.status === 401 ? "Invalid token." : e.message; }
}

document.querySelectorAll(".tabs button").forEach(b => b.addEventListener("click", () => {
  document.querySelectorAll(".tabs button").forEach(x => x.classList.toggle("active", x === b));
  $("tab-ledger").classList.toggle("hidden", b.dataset.tab !== "ledger");
  $("tab-funds").classList.toggle("hidden", b.dataset.tab !== "funds");
}));
$("tokenGo").addEventListener("click", () => tryToken($("tokenInput").value.trim()));
$("tokenInput").addEventListener("keydown", e => { if (e.key === "Enter") tryToken($("tokenInput").value.trim()); });
$("refreshLedger").addEventListener("click", loadLedger);
$("depGo").addEventListener("click", doDeposit);
$("wdGo").addEventListener("click", doWithdraw);
$("signOut").addEventListener("click", signOut);

if (token) tryToken(token);
</script>
</body>
</html>`
