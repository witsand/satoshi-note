# satoshi-note

A self-hosted Lightning Network voucher system. Create vouchers, fund them via LNURL-pay, and let recipients withdraw via LNURL-withdraw — all backed by the [Breez SDK Spark](https://github.com/breez/breez-sdk-spark) and a local SQLite database.

**How it works in plain English:**

1. You call the API to create one or more vouchers. Each voucher gets a unique secret and a public key.
2. Anyone with the public key can send sats to the voucher using any LNURL-compatible wallet.
3. Anyone with the secret can withdraw the balance using any LNURL-compatible wallet.
4. If a voucher expires before it is redeemed, the balance is automatically refunded to the address you set as the `refund_code`.

---

## Table of Contents

- [Requirements](#requirements)
- [Before You Begin](#before-you-begin)
- [Part 1 — Get a Virtual Server](#part-1--get-a-virtual-server)
- [Part 2 — Point Your Domain to the Server](#part-2--point-your-domain-to-the-server)
- [Part 3 — Prepare the Server](#part-3--prepare-the-server)
- [Part 4 — Install Go](#part-4--install-go)
- [Part 5 — Install and Configure Caddy](#part-5--install-and-configure-caddy)
- [Part 6 — Clone and Build satoshi-note](#part-6--clone-and-build-satoshi-note)
- [Part 7 — Configure the Application](#part-7--configure-the-application)
- [Part 8 — Run as a System Service](#part-8--run-as-a-system-service)
- [Part 9 — Wire Caddy to the App](#part-9--wire-caddy-to-the-app)
- [API Reference](#api-reference)
- [Configuration Reference](#configuration-reference)

---

## Requirements

- A domain name you control (e.g. `vouchers.example.com`)
- A Linux virtual server (VPS) running Ubuntu 22.04 or 24.04
- A [Breez API key](https://breez.technology) (free tier available)
- A BIP39 mnemonic (12 or 24 word seed phrase) — this is the wallet that holds sats in transit

---

## Before You Begin

**What is a mnemonic?**
A mnemonic is a seed phrase — a list of 12 or 24 words that represents a Bitcoin/Lightning wallet. Keep it secret and back it up. If you lose it, you lose access to any funds in the wallet. You can generate one with any BIP39 tool; for example [iancoleman.io/bip39](https://iancoleman.io/bip39) (use offline for safety, or just use a hardware wallet generator).

**What is a Breez API key?**
Breez SDK Spark is the Lightning Network backend this app uses. You need an API key from Breez to use it. Register at [breez.technology](https://breez.technology).

**What is Caddy?**
Caddy is a web server that automatically handles HTTPS for you — it gets and renews TLS certificates from Let's Encrypt with zero configuration. It sits in front of satoshi-note and forwards requests to it.

---

## Part 1 — Get a Virtual Server

You need a VPS running Ubuntu 22.04 or 24.04. Any provider works: [Hetzner](https://hetzner.com), [DigitalOcean](https://digitalocean.com), [Vultr](https://vultr.com), [Linode](https://linode.com), etc.

**Recommended minimum specs:**
- 1 vCPU, 2 GB RAM, 20 GB disk
- Ubuntu 22.04 or 24.04 LTS

When you create the server, note down the **public IP address** — you will need it in the next step.

Log in to the server via SSH. Your provider will give you instructions; it usually looks like:

```bash
ssh root@YOUR_SERVER_IP
```

Once logged in, update the system:

```bash
apt update && apt upgrade -y
```

---

## Part 2 — Point Your Domain to the Server

You need a domain name (or subdomain) pointing at your server's IP address. This is done by creating an **A record** in your domain registrar's DNS settings.

**Steps:**

1. Log in to wherever you bought your domain (Namecheap, Cloudflare, GoDaddy, etc.).
2. Find the DNS management page for your domain.
3. Create a new **A record**:

| Field | Value |
|---|---|
| **Type** | `A` |
| **Name / Host** | `vouchers` (or `@` for the root domain, or any subdomain you want) |
| **Value / Points to** | Your server's public IP address |
| **TTL** | `300` (5 minutes is fine) |

For example, if your domain is `example.com` and you set the name to `vouchers`, your app will be at `https://vouchers.example.com`.

> **Note:** DNS changes can take a few minutes to an hour to propagate. You can check if it has worked by running `ping vouchers.example.com` from your local machine — if it shows your server IP, you are ready.

---

## Part 3 — Prepare the Server

Create a dedicated user to run the application (running as root is not recommended):

```bash
useradd -m -s /bin/bash satoshi
```

Create the directory where the app and its data will live:

```bash
mkdir -p /opt/satoshi-note
chown satoshi:satoshi /opt/satoshi-note
```

---

## Part 4 — Install Go

Check the latest Go version at [go.dev/dl](https://go.dev/dl) and replace `1.24.3` below if a newer version is available.

```bash
cd /tmp
wget https://go.dev/dl/go1.24.3.linux-amd64.tar.gz
tar -C /usr/local -xzf go1.24.3.linux-amd64.tar.gz
```

Add Go to the system PATH so all users can use it:

```bash
echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
source /etc/profile
```

Verify it works:

```bash
go version
# Expected output: go version go1.24.3 linux/amd64
```

---

## Part 5 — Install and Configure Caddy

Caddy will handle HTTPS automatically and forward traffic to satoshi-note.

Install Caddy using the official method for Ubuntu:

```bash
apt install -y debian-keyring debian-archive-keyring apt-transport-https curl
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | tee /etc/apt/sources.list.d/caddy-stable.list
apt update
apt install caddy -y
```

Verify Caddy is installed:

```bash
caddy version
```

Enable and start Caddy so it runs on boot:

```bash
systemctl enable caddy
systemctl start caddy
```

You will finish configuring Caddy in [Part 9](#part-9--wire-caddy-to-the-app) once the app is running.

---

## Part 6 — Clone and Build satoshi-note

Switch to the satoshi user:

```bash
su - satoshi
```

Clone the repository:

```bash
cd /opt/satoshi-note
git clone https://github.com/YOUR_USERNAME/satoshi-note.git .
```

Build the binary:

```bash
go build -o satoshi-note .
```

This produces a single executable file called `satoshi-note` in the current directory. When done, exit back to root:

```bash
exit
```

---

## Part 7 — Configure the Application

The app is configured entirely through a `.env` file. Start from the example:

```bash
cp /opt/satoshi-note/.env.example /opt/satoshi-note/.env
```

Open the file for editing:

```bash
nano /opt/satoshi-note/.env
```

The file will look like this — fill in every value:

```env
# -------------------------------------------------------
# REQUIRED — you must set these four before the app starts
# -------------------------------------------------------
SITE_NAME=My Voucher Shop
BASE_URL=https://vouchers.example.com
MNEMONIC=word1 word2 word3 word4 word5 word6 word7 word8 word9 word10 word11 word12
BREEZ_API_KEY=your_breez_api_key_here

# -------------------------------------------------------
# Server
# -------------------------------------------------------
PORT=8080
NETWORK=mainnet
STORAGE_DIRECTORY=/opt/satoshi-note/.data

# -------------------------------------------------------
# Feature flags — set to false to disable an endpoint
# -------------------------------------------------------
CREATE_ACTIVE=true
FUND_ACTIVE=true
REDEEM_ACTIVE=true
REFUND_ACTIVE=true

# -------------------------------------------------------
# Invoice settings
# -------------------------------------------------------
INVOICE_EXPIRY_SECONDS=600

# -------------------------------------------------------
# Fee settings
# REDEEM_FEE_BPS: fee in basis points (50 = 0.5%)
# MIN_REDEEM_FEE_MSAT: minimum fee in millisatoshis
# -------------------------------------------------------
REDEEM_FEE_BPS=50
MIN_REDEEM_FEE_MSAT=10000

# -------------------------------------------------------
# Funding limits (in millisatoshis)
# 1 sat = 1000 msat
# 120000 msat = 120 sats
# 200000000 msat = 200,000 sats
# -------------------------------------------------------
MIN_FUND_AMOUNT_MSAT=120000
MAX_FUND_AMOUNT_MSAT=200000000
MIN_REDEEM_AMOUNT_MSAT=100000

# -------------------------------------------------------
# Voucher settings
# MAX_VOUCHER_EXPIRE_SECONDS: 31556952 = 1 year
# MAX_VOUCHERS_PER_BATCH: max vouchers in one create call
# RANDOM_BYTES_LENGTH: length of the secret (1-32)
# -------------------------------------------------------
MAX_VOUCHER_EXPIRE_SECONDS=31556952
MAX_VOUCHERS_PER_BATCH=100
RANDOM_BYTES_LENGTH=16
```

> **Important:** Replace `vouchers.example.com` with your actual domain. The `BASE_URL` must match exactly — this is how LNURL callbacks are built.

Save and close the file (`Ctrl+O`, `Enter`, `Ctrl+X` in nano).

Create the data directory and set correct ownership:

```bash
mkdir -p /opt/satoshi-note/.data
chown -R satoshi:satoshi /opt/satoshi-note
```

---

## Part 8 — Run as a System Service

Running the app as a `systemd` service means it will start automatically on boot and restart if it crashes.

Create the service file:

```bash
nano /etc/systemd/system/satoshi-note.service
```

Paste the following content:

```ini
[Unit]
Description=satoshi-note Lightning voucher server
After=network.target

[Service]
Type=simple
User=satoshi
WorkingDirectory=/opt/satoshi-note
EnvironmentFile=/opt/satoshi-note/.env
ExecStart=/opt/satoshi-note/satoshi-note
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Save and close the file, then reload systemd and start the service:

```bash
systemctl daemon-reload
systemctl enable satoshi-note
systemctl start satoshi-note
```

Check that it started correctly:

```bash
systemctl status satoshi-note
```

You should see `Active: active (running)`. To see the live logs at any time:

```bash
journalctl -u satoshi-note -f
```

The app is now running on port `8080` on `localhost`. It is not yet exposed to the internet — Caddy will do that next.

---

## Part 9 — Wire Caddy to the App

Now tell Caddy to listen on your domain, get a TLS certificate, and forward requests to the app.

Open the Caddy configuration file:

```bash
nano /etc/caddy/Caddyfile
```

Replace the entire contents with the following (substitute your real domain):

```
vouchers.example.com {
    reverse_proxy localhost:8080
}
```

That is the entire configuration. Caddy will automatically:
- Obtain a free TLS certificate from Let's Encrypt
- Renew it before it expires
- Redirect HTTP to HTTPS

Reload Caddy to apply the change:

```bash
systemctl reload caddy
```

Check Caddy's status:

```bash
systemctl status caddy
```

**Test it:** Open `https://vouchers.example.com` in a browser. You should see the static files served from the `static/` directory. If you see a padlock in the address bar, HTTPS is working.

---

## Verify Everything is Working

Run a quick end-to-end check by creating a test voucher:

```bash
curl -s -X POST https://vouchers.example.com/voucher/create \
  -H "Content-Type: application/json" \
  -d '{"amount": 1, "refund_code": "yourname@getalby.com", "refund_after_seconds": 86400}' | jq .
```

You should receive a JSON response with a `secret` and `pub_key`. The LNURL-pay link to fund the voucher would be:

```
https://vouchers.example.com/f/PUB_KEY_HERE
```

And the LNURL-withdraw link to redeem it:

```
https://vouchers.example.com/w/SECRET_HERE
```

These URLs can be encoded as LNURL bech32 strings for use with Lightning wallets that support LNURL.

---

## API Reference

All requests and responses use JSON. The server returns `{"status":"ERROR","reason":"..."}` for LNURL protocol errors.

---

### `POST /voucher/create`

Create one or more vouchers in a single batch.

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `amount` | integer | No | Number of vouchers to create (default: 1, max: `MAX_VOUCHERS_PER_BATCH`) |
| `batch_name` | string | No | Human-readable label for the batch |
| `refund_code` | string | No | Lightning address or LNURL to receive refunds when vouchers expire |
| `refund_after_seconds` | integer | Yes | Seconds until the voucher expires and balance is refunded |
| `single_use` | boolean | No | If `true`, voucher can only be redeemed once (default: `false`) |

**Example — create 3 single-use vouchers expiring after 7 days:**

```bash
curl -s -X POST https://vouchers.example.com/voucher/create \
  -H "Content-Type: application/json" \
  -d '{
    "amount": 3,
    "batch_name": "Event Giveaway",
    "refund_code": "you@getalby.com",
    "refund_after_seconds": 604800,
    "single_use": true
  }' | jq .
```

**Response** `201 Created` — JSON array of voucher objects:

```json
[
  {
    "id": 1,
    "secret": "a3f1c2d4e5b6a7f8c9d0e1f2a3b4c5d6",
    "pub_key": "3f1c2d4e5b6a7f8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2",
    "batch_name": "Event Giveaway",
    "batch_id": "c1d2e3f4a5b6c7d8",
    "refund_code": "you@getalby.com",
    "refund_after_seconds": 604800,
    "balance_msat": 0,
    "single_use": true
  }
]
```

---

### LNURL-pay (Funding)

These endpoints follow the [LUD-06](https://github.com/lnurl/luds/blob/legacy/lnurl-pay.md) LNURL-pay spec.

| Method | Path | Description |
|---|---|---|
| `GET` | `/f/{pubKey}` | LNURL-pay step 1 — fund a single voucher by its public key |
| `GET` | `/fb/{batchID}` | LNURL-pay step 1 — fund all vouchers in a batch equally |
| `GET` | `/fund/{pubKey}/callback` | LNURL-pay step 2 callback for single voucher |
| `GET` | `/fund/batch/{batchID}/callback` | LNURL-pay step 2 callback for batch |
| `GET` | `/fv/{key}` | LUD-21 payment verify endpoint |

To fund a voucher, encode `https://vouchers.example.com/f/{pubKey}` as a bech32 LNURL string and scan it with a compatible wallet (e.g. Phoenix, Zeus, Wallet of Satoshi).

---

### LNURL-withdraw (Redemption)

These endpoints follow the [LUD-03](https://github.com/lnurl/luds/blob/legacy/lnurl-withdraw.md) LNURL-withdraw spec.

| Method | Path | Description |
|---|---|---|
| `GET` | `/w/{secret}` | LNURL-withdraw step 1 — initiate redemption with the voucher secret |
| `GET` | `/withdraw/{secret}/callback` | LNURL-withdraw step 2 callback |

To redeem a voucher, encode `https://vouchers.example.com/w/{secret}` as a bech32 LNURL string and scan it with a compatible wallet.

---

## Configuration Reference

All variables are read from the `.env` file. Variables marked **Required** will cause the app to refuse to start if missing.

| Variable | Required | Default | Description |
|---|---|---|---|
| `SITE_NAME` | Yes | — | Display name shown in LNURL metadata |
| `BASE_URL` | Yes | — | Public HTTPS URL of the server (no trailing slash) |
| `MNEMONIC` | Yes | — | BIP39 seed phrase for the Lightning wallet |
| `BREEZ_API_KEY` | Yes | — | API key from Breez SDK |
| `PORT` | Yes | `8080` | Port the HTTP server listens on |
| `NETWORK` | No | `regtest` | `mainnet` or `regtest` |
| `STORAGE_DIRECTORY` | Yes | `./.data` | Directory for the SQLite database file |
| `CREATE_ACTIVE` | No | `true` | Enable/disable voucher creation endpoint |
| `FUND_ACTIVE` | No | `true` | Enable/disable LNURL-pay funding endpoints |
| `REDEEM_ACTIVE` | No | `true` | Enable/disable LNURL-withdraw redemption endpoints |
| `REFUND_ACTIVE` | No | `true` | Enable/disable automatic refund worker |
| `INVOICE_EXPIRY_SECONDS` | Yes | `600` | How long generated invoices are valid (seconds) |
| `REDEEM_FEE_BPS` | Yes | `50` | Service fee in basis points (50 = 0.5%) |
| `MIN_REDEEM_FEE_MSAT` | Yes | `10000` | Minimum service fee in millisatoshis |
| `MIN_FUND_AMOUNT_MSAT` | Yes | `120000` | Minimum amount a voucher can be funded (millisatoshis) |
| `MAX_FUND_AMOUNT_MSAT` | Yes | `200000000` | Maximum amount a voucher can be funded (millisatoshis) |
| `MIN_REDEEM_AMOUNT_MSAT` | Yes | `100000` | Minimum redeemable balance (millisatoshis) |
| `MAX_VOUCHER_EXPIRE_SECONDS` | Yes | `31556952` | Maximum allowed `refund_after_seconds` value (1 year) |
| `MAX_VOUCHERS_PER_BATCH` | Yes | `100` | Maximum number of vouchers per create request |
| `RANDOM_BYTES_LENGTH` | Yes | `16` | Byte length of generated secrets (1–32) |

---

## Troubleshooting

**The app won't start — "missing environment variable"**
Open your `.env` file and make sure every required variable is set and uncommented (no `#` at the start of the line).

**HTTPS isn't working / Caddy shows an error**
Make sure your DNS A record is pointing to the correct IP and has had time to propagate. Run `ping yourdomain.com` to confirm. Also ensure ports 80 and 443 are open in your server's firewall:
```bash
ufw allow 80
ufw allow 443
ufw allow 22
ufw enable
```

**The app starts but Lightning payments fail**
Check the logs with `journalctl -u satoshi-note -f`. Common causes: incorrect `BREEZ_API_KEY`, wrong `NETWORK` setting, or insufficient on-chain funds for the Spark wallet to open a channel.

**How do I update the app?**
```bash
cd /opt/satoshi-note
su - satoshi -c "cd /opt/satoshi-note && git pull && go build -o satoshi-note ."
systemctl restart satoshi-note
```

---

## License

GNU General Public License v3.0 — see [LICENSE](LICENSE).
