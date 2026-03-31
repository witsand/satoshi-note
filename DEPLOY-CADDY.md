# Satoshi Note — Deployment Guide (Caddy + Direct DNS)

This guide covers deploying Satoshi Note on a VPS (**LunaNode** or **MyNymBox**) using **Caddy** as the reverse proxy with automatic HTTPS, and a domain registered with either **GoDaddy** or **MyNymBox**.

Unlike the Cloudflare Tunnel approach in README.md, this setup routes traffic directly to your server — Caddy handles TLS certificates automatically via Let's Encrypt.

---

## What you'll need before starting

- A VPS on LunaNode or MyNymBox (see Step 1)
- A domain name from GoDaddy or MyNymBox (see Step 2)
- A [Breez API key](https://breez.technology) — sign up and request one
- A BIP39 seed phrase (12–24 words) for your Lightning wallet — generate one offline at [iancoleman.io/bip39](https://iancoleman.io/bip39)

---

## Step 1 — Create a VPS

### Option A: LunaNode

LunaNode accepts Bitcoin payments and offers straightforward Linux VMs.

1. Go to [lunanode.com](https://lunanode.com) and create an account
2. Add credit (Bitcoin accepted via the billing panel)
3. Go to **Virtual Machines** → **Launch VM**
   - **Image:** Ubuntu 24.04 LTS
   - **Flavor:** `m1.small` or larger — minimum 1 vCPU, 2 GB RAM
   - **Region:** pick one geographically close to you
   - **Keypair:** add your SSH public key, or note the root password shown after launch
4. After launch, note the **public IPv4 address** shown in the VM list

SSH in:
```bash
ssh ubuntu@YOUR_IP
```

Update the system:
```bash
sudo apt update && sudo apt upgrade -y
```

### Option B: MyNymBox

MyNymBox is a privacy-focused hosting provider that accepts crypto.

1. Go to [mynymbox.com](https://mynymbox.com) and register an account
2. Purchase a VPS plan — minimum 1 GB RAM, 1 vCPU is sufficient
3. During provisioning, select **Ubuntu 24.04** as your OS
4. Once the VM is active, note the **public IPv4 address** from your dashboard
5. Retrieve your SSH credentials from the dashboard (root password or SSH key option)

SSH in:
```bash
ssh root@YOUR_IP
```

> If you're on MyNymBox as root, replace `ubuntu` with `root` in all paths below, or create a non-root user:
> ```bash
> adduser satoshi
> usermod -aG sudo satoshi
> su - satoshi
> ```

Update the system:
```bash
sudo apt update && sudo apt upgrade -y
```

---

## Step 2 — Point your domain to the server

You need an **A record** pointing your domain at the VPS's public IP. The process differs by registrar.

### Option A: GoDaddy

1. Log in to [godaddy.com](https://godaddy.com) → **My Products** → click **DNS** next to your domain
2. In the DNS records table, find the **A** record for `@` (the root domain)
3. Click the pencil icon to edit it — set the **Value** to your VPS's public IP address
4. Set TTL to **600** (10 minutes) — makes propagation faster during setup
5. Click **Save**
6. If you want `www` to also work, add or edit the `www` CNAME record to point to `@`

To verify propagation (run this from your local machine — may take a few minutes to 1 hour):
```bash
nslookup yourdomain.com
# or
dig +short yourdomain.com
```
It should return your VPS IP.

### Option B: MyNymBox domain

If you registered the domain through MyNymBox, DNS is managed in their control panel.

1. Log in → go to **Domains** → click your domain → **DNS Management** (or **Zone Editor**)
2. Find or create an **A record**:
   - **Name/Host:** `@` (represents the root domain)
   - **Type:** A
   - **Value/Points to:** your VPS public IP
   - **TTL:** 600
3. If you want `www` to resolve as well, add a CNAME record:
   - **Name:** `www`
   - **Type:** CNAME
   - **Value:** `yourdomain.com.` (with the trailing dot)
4. Save and allow time for propagation (usually under 30 minutes)

---

## Step 3 — Open firewall ports on the VPS

Caddy needs ports **80** (HTTP for ACME challenge) and **443** (HTTPS) reachable from the internet.

### LunaNode

1. In the LunaNode dashboard, go to **Networking** → **Security Groups**
2. Add **inbound** rules for:
   - Port **80**, protocol TCP, source `0.0.0.0/0`
   - Port **443**, protocol TCP, source `0.0.0.0/0`
3. Attach the security group to your VM if it isn't already

### MyNymBox

MyNymBox VMs typically have no external firewall by default — all ports open. Confirm by checking your control panel under **Firewall** or **Network**.

### UFW (on the VM itself)

Regardless of provider, configure the host firewall:
```bash
sudo ufw allow OpenSSH
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable
sudo ufw status
```

---

## Step 4 — Install Go and build the app

```bash
# Install build tools (required for CGO — used by Breez SDK and SQLite)
sudo apt install -y build-essential git

# Install Go 1.24
cd /tmp
wget https://go.dev/dl/go1.24.3.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.24.3.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
go version
# Expected: go version go1.24.3 linux/amd64
```

Clone and build:
```bash
cd ~
git clone https://github.com/witsand/satoshi-note.git send-sats
cd send-sats
go build -o send-sats .
```

Build takes 1–3 minutes on first run (downloads and compiles dependencies). The output binary is `./send-sats`.

---

## Step 5 — Configure the app

```bash
cp .env.example .env
nano .env
```

Fill in the values below. Lines marked **REQUIRED** must be set; others can stay at their defaults.

```env
# REQUIRED — branding (exactly two words)
SITE_NAME="Send Sats"
SITE_NAME_ORANGE_WORD=2

# REQUIRED — your actual domain with https://
BASE_URL=https://yourdomain.com

# REQUIRED — your Lightning wallet seed phrase (keep this secret and backed up)
MNEMONIC="word1 word2 word3 word4 word5 word6 word7 word8 word9 word10 word11 word12"

# REQUIRED — from breez.technology
BREEZ_API_KEY=your_breez_api_key_here

# OPTIONAL — a long random string to access /admin/stats and /admin/recent
# Leave blank to disable admin endpoints
ADMIN_TOKEN=a_long_random_string_here

# Server
PORT=8080
NETWORK=mainnet
STORAGE_DIRECTORY=/home/ubuntu/send-sats/.data

# Optional — shows a GitHub link in the About modal
# GITHUB_URL=https://github.com/witsand/satoshi-note

# Feature flags — set to true to enable
CREATE_ACTIVE=true
FUND_ACTIVE=true
REDEEM_ACTIVE=true
REFUND_ACTIVE=true
REFUND_WORKER_INTERVAL_SECONDS=14400
PAYMENT_COOLDOWN_MS=200

# Frontend
BATCH_ENABLED=false
DEFAULT_DIAL_CODE=+1

# Invoice expiry
INVOICE_EXPIRY_SECONDS=300

# Fees
REDEEM_FEE_BPS=50
MIN_REDEEM_FEE_MSAT=10000
INTERNAL_FEE_BPS=5
MIN_INTERNAL_FEE_MSAT=1000

# Voucher limits
MIN_FUND_AMOUNT_MSAT=110000
MAX_FUND_AMOUNT_MSAT=1000000000
MIN_REDEEM_AMOUNT_MSAT=100000
MAX_VOUCHER_EXPIRE_SECONDS=31556952

# Voucher creation
RANDOM_BYTES_LENGTH=32
MAX_VOUCHERS_PER_BATCH=32
```

Save and exit: `Ctrl+O`, `Enter`, `Ctrl+X`.

> **Security:** restrict `.env` permissions so only your user can read it:
> ```bash
> chmod 600 .env
> ```

Create the data directory:
```bash
mkdir -p .data
```

If you're running as root (MyNymBox), adjust `STORAGE_DIRECTORY` accordingly:
```env
STORAGE_DIRECTORY=/root/send-sats/.data
```

---

## Step 6 — Run the app as a systemd service

```bash
sudo nano /etc/systemd/system/send-sats.service
```

Paste the following (adjust `User` and paths if you're running as `root`):

```ini
[Unit]
Description=Satoshi Note Lightning voucher server
After=network.target

[Service]
Type=simple
User=ubuntu
WorkingDirectory=/home/ubuntu/send-sats
EnvironmentFile=/home/ubuntu/send-sats/.env
ExecStart=/home/ubuntu/send-sats/send-sats
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

> If you're on MyNymBox as `root`, use `User=root` and `/root/send-sats/` for the paths.

Enable and start:
```bash
sudo systemctl daemon-reload
sudo systemctl enable send-sats
sudo systemctl start send-sats
sudo systemctl status send-sats
```

You should see `active (running)`. If not, check logs:
```bash
journalctl -u send-sats -f
```

Verify the app is responding locally:
```bash
curl http://localhost:8080/config
# Should return a JSON object with your configuration
```

---

## Step 7 — Install and configure Caddy

Caddy automatically obtains and renews TLS certificates via Let's Encrypt. No manual certificate management needed.

### Install Caddy

```bash
sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https curl
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' \
  | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' \
  | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update
sudo apt install -y caddy
caddy version
```

### Configure Caddy

Open the Caddyfile:
```bash
sudo nano /etc/caddy/Caddyfile
```

Replace the entire contents with the following (substitute your actual domain):

```caddy
yourdomain.com {
    reverse_proxy localhost:8080 {
        header_up X-Real-IP {remote_host}
    }
}
```

> **Why `header_up X-Real-IP {remote_host}`?**
> Satoshi Note's rate limiter reads the real client IP from `X-Real-IP` (or Cloudflare's `CF-Connecting-IP`). Without this header, all requests appear to come from `127.0.0.1` and the rate limiter treats every visitor as the same client.
> Do **not** use `X-Forwarded-For` — its leftmost value is client-controlled and can be spoofed.

If you also want `www.yourdomain.com` to redirect to the root:
```caddy
yourdomain.com {
    reverse_proxy localhost:8080 {
        header_up X-Real-IP {remote_host}
    }
}

www.yourdomain.com {
    redir https://yourdomain.com{uri} permanent
}
```

Validate the config:
```bash
caddy validate --config /etc/caddy/Caddyfile
# Should print: Valid configuration
```

Reload Caddy:
```bash
sudo systemctl reload caddy
sudo systemctl status caddy
```

Caddy will immediately begin requesting a TLS certificate from Let's Encrypt. This requires:
- Your domain's A record already pointing to this server's IP
- Port 80 reachable from the internet (for the ACME HTTP-01 challenge)

Certificate issuance usually completes within 30 seconds.

---

## Step 8 — Verify the deployment

Open `https://yourdomain.com` in a browser. You should see the Satoshi Note interface with a valid TLS certificate (padlock icon).

Test the API directly:
```bash
curl https://yourdomain.com/config
```

Check that both services survive a reboot:
```bash
sudo reboot
```

After ~30 seconds, SSH back in:
```bash
ssh ubuntu@YOUR_IP
sudo systemctl status send-sats caddy
```

Both should show `active (running)`.

---

## Step 9 — Confirm Caddy certificate auto-renewal

Caddy renews certificates automatically in the background — no cron job needed. You can confirm it's working:

```bash
# View certificate information
caddy trust
# or check Caddy's managed certificates
journalctl -u caddy | grep -i "certificate\|acme\|tls"
```

Caddy renews certificates when they have ~30 days remaining. Nothing else to configure.

---

## Updating the app

```bash
cd ~/send-sats
git pull
go build -o send-sats .
sudo systemctl restart send-sats
sudo systemctl status send-sats
```

Caddy does not need to be restarted for app updates.

---

## Troubleshooting

**`journalctl -u send-sats -f` shows the app crashing on start:**
- Check your `.env` — all required fields must be set
- Confirm `STORAGE_DIRECTORY` exists and is writable: `mkdir -p /home/ubuntu/send-sats/.data`
- Confirm `NETWORK` is `mainnet` or `regtest` (anything other than `mainnet` → regtest)
- Confirm `SITE_NAME` is exactly two words

**Caddy fails to get a certificate:**
- Confirm DNS propagation: `dig +short yourdomain.com` must return your server IP
- Confirm port 80 is open: `sudo ufw status`, check your provider's security group
- Check Caddy logs: `journalctl -u caddy -f`

**`https://yourdomain.com` shows a connection error:**
- Check that the app is running: `curl http://localhost:8080/config`
- Check that Caddy is running: `sudo systemctl status caddy`
- Check the Caddyfile port matches `PORT` in your `.env`

**Rate limiter treating all users as one client:**
- Confirm your Caddyfile includes `header_up X-Real-IP {remote_host}` inside the `reverse_proxy` block

**Breez SDK fails to connect:**
- Breez SDK requires an outbound internet connection on startup
- Confirm the VM can reach the internet: `curl https://breez.technology`
- Breez SDK initial sync can take 30–60 seconds — watch `journalctl -u send-sats -f`

---

## File locations summary

| File | Path |
|------|------|
| App binary | `/home/ubuntu/send-sats/send-sats` |
| Environment config | `/home/ubuntu/send-sats/.env` |
| SQLite database | `/home/ubuntu/send-sats/.data/satoshi_note.db` |
| Breez wallet data | `/home/ubuntu/send-sats/.data/mainnet/` |
| Systemd unit | `/etc/systemd/system/send-sats.service` |
| Caddyfile | `/etc/caddy/Caddyfile` |
| Caddy TLS certs | `/var/lib/caddy/.local/share/caddy/` |

---

## Security reminders

- **Back up your `MNEMONIC`** — it controls all Lightning wallet funds. Store it offline in multiple locations. Losing it means losing all unclaimed voucher balances.
- **Keep `.env` private** — it contains your mnemonic, API key, and admin token. `chmod 600 .env` and do not commit it to version control.
- **Do not expose port 8080** — the app speaks plain HTTP. Only Caddy (on 443) should be publicly reachable. UFW already blocks 8080 if configured per Step 3.
- **`ADMIN_TOKEN`** — if set, anyone who knows it can view stats and recent transactions. Use a long random string (e.g. `openssl rand -hex 32`).
