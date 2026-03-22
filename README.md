# Satoshi Note

A self-hosted Lightning voucher system. Create vouchers, fund them via LNURL-pay, and share a claim link with the recipient — who withdraws via LNURL-withdraw. Backed by Breez SDK Spark and SQLite.

How it works: create a voucher → share the funding link → fund it → recipient receives a claim link → they withdraw the sats. If unclaimed before expiry, the sats auto-refund to your Lightning address.

---

## Before you start

- A domain name managed by Cloudflare (or you'll point its nameservers to Cloudflare in Step 2)
- A [Breez API key](https://breez.technology)
- A BIP39 seed phrase (12 words) for the Lightning wallet — generate one at [iancoleman.io/bip39](https://iancoleman.io/bip39) (do it offline)
- A Cloudflare account (free)

---

## Step 1 — Create a VM on LunaNode

- Go to [lunanode.com](https://lunanode.com) — you can pay with Bitcoin
- Create account → Launch VM
- **Image:** Ubuntu 24.04
- **Recommended size:** m1.small or larger (1 vCPU, 2 GB RAM)
- **Region:** pick one near you
- After launch: note the public IP and the SSH password LunaNode shows you

SSH in:
```bash
ssh ubuntu@{YOUR_IP}
```

Update the system:
```bash
sudo apt update && sudo apt upgrade -y
```

---

## Step 2 — Set up your domain on Cloudflare

**If you don't have a Cloudflare account yet:**
- Sign up at [cloudflare.com](https://cloudflare.com) → Add a Site → enter your domain
- Cloudflare will show you two nameservers (e.g. `aria.ns.cloudflare.com`)
- Log in to your domain registrar → change nameservers to the ones Cloudflare gave you
- Wait for propagation (can take up to 24h but usually ~15 min)

**If your domain is already on Cloudflare:** nothing to do here.

No DNS record is needed — cloudflared will create it automatically in Step 7.

---

## Step 3 — Install Go and build the app

```bash
# Install Go
cd /tmp
wget https://go.dev/dl/go1.24.3.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.24.3.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
go version

# Install C build tools (required for CGO — used by Breez SDK)
sudo apt install -y build-essential

# Clone and build
cd ~
git clone https://github.com/witsand/satoshi-note.git send-sats
cd send-sats
go build -o send-sats .
```

---

## Step 4 — Configure the app

```bash
cp .env.example .env
nano .env
```

Fill in the required values:

```env
SITE_NAME="Send Sats"
SITE_NAME_ORANGE_WORD=2
BASE_URL=https://yourdomain.com
MNEMONIC="word1 word2 ... word12"
BREEZ_API_KEY=your_key_here
PORT=8080
NETWORK=mainnet
STORAGE_DIRECTORY=/home/ubuntu/send-sats/.data
```

Leave everything else at its default. Save with `Ctrl+O`, `Enter`, `Ctrl+X`.

```bash
mkdir -p .data
```

---

## Step 5 — Run the app as a system service

```bash
sudo nano /etc/systemd/system/send-sats.service
```

Paste:

```ini
[Unit]
Description=Send Sats Lightning voucher server
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

```bash
sudo systemctl daemon-reload
sudo systemctl enable send-sats
sudo systemctl start send-sats
sudo systemctl status send-sats   # should show: active (running)
```

Check logs: `journalctl -u send-sats -f`

---

## Step 6 — Install cloudflared

```bash
curl -L --output /tmp/cloudflared.deb \
  https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64.deb
sudo dpkg -i /tmp/cloudflared.deb
cloudflared --version
```

---

## Step 7 — Create and configure the tunnel

```bash
# Authenticate — this prints a URL. Open it in your browser and log in to Cloudflare.
cloudflared tunnel login

# Create the tunnel (gives you a UUID)
cloudflared tunnel create send-sats

# Write the config file (replace <UUID> with your tunnel UUID from above)
mkdir -p ~/.cloudflared
cat > ~/.cloudflared/config.yml << 'EOF'
tunnel: <UUID>
credentials-file: /home/ubuntu/.cloudflared/<UUID>.json

ingress:
  - hostname: yourdomain.com
    service: http://localhost:8080
  - service: http_status:404
EOF

# Create the DNS CNAME record in Cloudflare automatically
cloudflared tunnel route dns send-sats yourdomain.com

# Test it manually first (Ctrl+C to stop)
cloudflared tunnel run send-sats
```

At this point visiting `https://yourdomain.com` should show the app.

---

## Step 8 — Auto-start cloudflared on reboot

```bash
sudo nano /etc/systemd/system/cloudflared.service
```

Paste:

```ini
[Unit]
Description=Cloudflare Tunnel
After=network.target

[Service]
Type=simple
User=ubuntu
ExecStart=/usr/bin/cloudflared tunnel --config /home/ubuntu/.cloudflared/config.yml run
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable cloudflared
sudo systemctl start cloudflared
sudo systemctl status cloudflared   # should show: active (running)
```

---

## Step 9 — Verify

Open `https://yourdomain.com` in a browser — you should see the app.

After a reboot, wait ~30 seconds, SSH back in and confirm both services are running:

```bash
sudo reboot
# ... reconnect after ~30s ...
sudo systemctl status send-sats cloudflared
```

---

## Updating the app

```bash
cd ~/send-sats
git pull
go build -o send-sats .
sudo systemctl restart send-sats
```

---

## Notes

- No firewall rules needed — cloudflared tunnel is outbound only; the VM never needs ports 80/443 open
- Cloudflare handles TLS automatically via the tunnel
- The `.env` file contains secrets — keep it private
- Keep a backup of your `MNEMONIC` somewhere safe — it holds your Lightning wallet funds

---

## Running behind nginx or caddy instead of Cloudflare Tunnel

The rate limiter reads the real client IP from the `CF-Connecting-IP` header (set by Cloudflare) or `X-Real-IP` (set by nginx/caddy). If neither header is present it falls back to the raw TCP connection address. You **must** configure your proxy to set `X-Real-IP`, otherwise all traffic appears to come from `127.0.0.1` and the rate limiter treats the entire world as one client.

**nginx** — add inside your `location /` block:
```nginx
proxy_set_header X-Real-IP $remote_addr;
```

**caddy** — add inside your `reverse_proxy` block:
```caddy
header_up X-Real-IP {remote_host}
```

Do not forward `X-Forwarded-For` as a substitute — its leftmost value is client-controlled and can be spoofed to bypass rate limiting.
