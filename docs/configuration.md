# Configuration Reference

All configuration is done through environment variables in the `.env` file. Copy `.env.example` to `.env` before editing.

```bash
cp .env.example .env
```

---

## Required Variables

These must be set before Joules will start.

### `DATABASE_URL`

PostgreSQL connection string.

```env
DATABASE_URL=postgres://joule:joule@db:5432/joule?sslmode=disable
```

If you're using the bundled `db` container (default), leave this value as-is. The `db` in the hostname refers to the Docker service name.

If you're connecting to an external PostgreSQL server:
```env
DATABASE_URL=postgres://myuser:mypassword@192.168.1.50:5432/joules?sslmode=require
```

### `JWT_SECRET`

A secret string used to sign authentication tokens. Use anything long and random — nobody should ever guess this.

```bash
# Generate a good one:
openssl rand -hex 32
```

```env
JWT_SECRET=a3f8c2e1d9b4...
```

### `AI_PROVIDER`

Which AI provider to use for food photo analysis and the health coach.

```env
AI_PROVIDER=openai      # Use OpenAI (default)
AI_PROVIDER=anthropic   # Use Anthropic Claude
```

One of `OPENAI_API_KEY` or `ANTHROPIC_API_KEY` must also be set — see below.

---

## AI Provider Variables

### `OPENAI_API_KEY`

Required when `AI_PROVIDER=openai`.

```env
OPENAI_API_KEY=sk-proj-...
```

### `ANTHROPIC_API_KEY`

Required when `AI_PROVIDER=anthropic`.

```env
ANTHROPIC_API_KEY=sk-ant-...
```

### `AI_MODEL`

Override the default model. If not set, Joules uses the recommended default for your provider.

| Provider | Default model |
|----------|--------------|
| openai | `gpt-4o` |
| anthropic | `claude-sonnet-4-6` |

Examples:
```env
AI_MODEL=gpt-4o-mini                    # Cheaper/faster OpenAI
AI_MODEL=claude-haiku-4-5-20251001      # Cheaper/faster Anthropic
```

See [AI Providers](ai-providers.md) for a full model comparison.

### `OPENAI_BASE_URL`

Override the OpenAI API endpoint. Useful for using OpenAI-compatible providers (e.g. a local Ollama instance, Azure OpenAI, OpenRouter).

```env
OPENAI_BASE_URL=http://localhost:11434/v1   # Ollama
OPENAI_BASE_URL=https://openrouter.ai/api/v1
```

---

## Optional Variables

### `PORT`

The port Joules listens on inside the container.

```env
PORT=8687   # default
```

If you change this, also update the port mapping in `docker-compose.yml`.

---

## Email Variables

Without SMTP configured, verification codes are printed to Docker logs. See [Email Setup](email-setup.md) for full details.

### `SMTP_HOST`

```env
SMTP_HOST=smtp.gmail.com
SMTP_HOST=mail.yourdomain.com
```

### `SMTP_PORT`

```env
SMTP_PORT=465   # Implicit TLS (recommended)
SMTP_PORT=587   # STARTTLS
```

### `SMTP_USER`

The email address used as the sender.

```env
SMTP_USER=hello@yourdomain.com
```

### `SMTP_PASS`

The SMTP account password or app password.

```env
SMTP_PASS=your-app-password
```

---

## Example `.env`

```env
# Database
DATABASE_URL=postgres://joule:joule@db:5432/joule?sslmode=disable

# Security
JWT_SECRET=a3f8c2e1d9b47f6e8a2c1d0e9f3b5a7c4d8e2f1a6b9c3d5e7f0a2b4c6d8e0f1

# AI
AI_PROVIDER=openai
OPENAI_API_KEY=sk-proj-...
# AI_MODEL=gpt-4o-mini   # Uncomment to use a cheaper model

# Email (optional)
# SMTP_HOST=smtp.gmail.com
# SMTP_PORT=587
# SMTP_USER=you@gmail.com
# SMTP_PASS=your-app-password
```
