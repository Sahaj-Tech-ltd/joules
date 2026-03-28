# Getting Started

This guide walks you through setting up Joules from scratch. No prior experience with Docker or self-hosting is required.

## Prerequisites

You need two things installed on your server or computer:

- **Docker** — the container runtime
- **Docker Compose** — included with Docker Desktop; on Linux, install separately

### Install Docker

**Ubuntu/Debian:**
```bash
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
# Log out and back in for the group change to take effect
```

**macOS / Windows:** Download and install [Docker Desktop](https://www.docker.com/products/docker-desktop/).

Verify the installation:
```bash
docker --version
docker compose version
```

### Get an AI API Key

Joules uses AI to identify food from photos and power the health coach. You need a key from one of the supported providers:

- **OpenAI** — [platform.openai.com/api-keys](https://platform.openai.com/api-keys) — pay-per-use, starts cheap
- **Anthropic** — [console.anthropic.com](https://console.anthropic.com) — pay-per-use, starts cheap

You only need one. See [AI Providers](ai-providers.md) for a comparison.

---

## Installation

### 1. Download Joules

```bash
git clone https://github.com/Sahaj-Tech-ltd/joules.git
cd joules
```

No Git? Download the ZIP from the GitHub page and extract it.

### 2. Create Your Config File

```bash
cp .env.example .env
```

Open `.env` in any text editor and fill in the required fields:

```env
# ── AI Provider ──────────────────────────────────────────
# Choose one: openai or anthropic
AI_PROVIDER=openai
OPENAI_API_KEY=sk-...
# ANTHROPIC_API_KEY=sk-ant-...

# ── Database ─────────────────────────────────────────────
# Leave this as-is unless you're using an external database
DATABASE_URL=postgres://joule:joule@db:5432/joule?sslmode=disable

# ── Security ─────────────────────────────────────────────
# Generate a random string — anything long and unguessable works
# Example: openssl rand -hex 32
JWT_SECRET=replace-this-with-a-random-string
```

Only these three fields are required to get started. Everything else is optional.

### 3. Start the App

```bash
docker compose up --build
```

The first run downloads and builds everything — this takes a few minutes. You'll see a stream of log output. When you see something like `server listening on :3000`, it's ready.

Open your browser and go to: **http://localhost:3000**

> **Running on a remote server?** Use the server's IP address instead: `http://YOUR_SERVER_IP:3000`. Make sure port 3000 is open in your firewall. To expose it on the internet with a domain name, see [Reverse Proxy](reverse-proxy.md).

### 4. Create Your Account

1. Click **Sign Up** and enter an email address and password
2. Since email isn't configured yet, your verification code is in the Docker logs:
   ```bash
   docker compose logs joule | grep -i verification
   ```
3. Enter the code on the verification page
4. Complete the onboarding wizard — enter your height, weight, age, activity level, and diet goal
5. You're in!

> **Want real email verification?** See [Email Setup](email-setup.md).

---

## Running in the Background

The command above keeps logs printing to your terminal. To run Joules as a background service:

```bash
docker compose up -d
```

Manage it with:
```bash
docker compose logs -f      # Stream logs
docker compose stop         # Stop containers (keeps data)
docker compose down         # Stop and remove containers (keeps data)
docker compose down -v      # Stop, remove containers AND data (destructive!)
```

---

## Updating Joules

```bash
docker compose pull
docker compose up -d --build
```

---

## Next Steps

- [Usage Guide](usage-guide.md) — how to log meals, track workouts, and use the AI coach
- [Configuration](configuration.md) — all available settings
- [Reverse Proxy](reverse-proxy.md) — access Joules from the internet with a domain name
