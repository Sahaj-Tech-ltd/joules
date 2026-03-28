# Reverse Proxy

By default, Joules runs on `http://localhost:8687` (or whatever port you set). To access it from the internet with a domain name and HTTPS, you need a reverse proxy.

This guide covers two options: **Caddy** (recommended for simplicity) and **Nginx**.

---

## Prerequisites

- A domain name with DNS pointing to your server's public IP
- Ports 80 and 443 open in your firewall

---

## Option 1: Caddy (Recommended)

Caddy automatically handles HTTPS certificate provisioning via Let's Encrypt. No manual certificate management needed.

### Install Caddy

**Ubuntu/Debian:**
```bash
sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https curl
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update && sudo apt install caddy
```

### Configure Caddy

Edit `/etc/caddy/Caddyfile`:

```caddyfile
joules.yourdomain.com {
    reverse_proxy localhost:8687
}
```

Apply the config:
```bash
sudo systemctl reload caddy
```

That's it. Caddy will obtain and renew the TLS certificate automatically.

### Running Caddy in Docker (alongside Joules)

If you prefer a fully Docker-based setup, add Caddy as a service in `docker-compose.yml`:

```yaml
services:
  caddy:
    image: caddy:2-alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
      - caddy_data:/data
      - caddy_config:/config
    depends_on:
      - joule

volumes:
  caddy_data:
  caddy_config:
```

`Caddyfile` (in the same directory as `docker-compose.yml`):
```caddyfile
joules.yourdomain.com {
    reverse_proxy joule:8687
}
```

---

## Option 2: Nginx

### Install Nginx

```bash
sudo apt update && sudo apt install -y nginx certbot python3-certbot-nginx
```

### Configure Nginx

Create `/etc/nginx/sites-available/joules`:

```nginx
server {
    listen 80;
    server_name joules.yourdomain.com;

    location / {
        proxy_pass http://localhost:8687;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # Required for photo uploads
        client_max_body_size 10M;

        # WebSocket support (for future use)
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

Enable and reload:
```bash
sudo ln -s /etc/nginx/sites-available/joules /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

### Enable HTTPS with Let's Encrypt

```bash
sudo certbot --nginx -d joules.yourdomain.com
```

Certbot will automatically configure HTTPS and set up certificate renewal.

---

## Using Cloudflare Tunnel

If your server doesn't have a public IP (behind NAT, residential connection), Cloudflare Tunnel is a good alternative — no open ports required.

After installing `cloudflared` and authenticating:

```bash
cloudflared tunnel create joules
cloudflared tunnel route dns joules joules.yourdomain.com
```

Configure `~/.cloudflared/config.yml`:
```yaml
tunnel: joules
credentials-file: /root/.cloudflared/<tunnel-id>.json

ingress:
  - hostname: joules.yourdomain.com
    service: http://localhost:8687
  - service: http_status:404
```

Run:
```bash
cloudflared tunnel run joules
```

Or as a systemd service:
```bash
sudo cloudflared service install
sudo systemctl start cloudflared
```
