# Joules — Self-Hosted AI Calorie Tracker

**Joules** is a privacy-first, self-hosted calorie tracking app. Snap a photo of your meal, and AI identifies the food and estimates calories/macros. Track progress against personalized diet plans with an AI health coach.

> **New to Joules?** Start with the [Getting Started guide](docs/getting-started.md) — no experience required.

## Why Self-Hosted?

- **No paywalls** — You own your data, no subscriptions required
- **Complete control** — Choose your AI model (OpenAI GPT-4, Anthropic Claude, or compatible endpoints)
- **Privacy** — Your food photos and health data stay on your server
- **Free forever** — One Docker container, PostgreSQL database, done

## Features

- **Photo-based meal logging** — AI identifies food and estimates nutrition
- **Manual food entry** — Add foods manually with calorie/macro details
- **8 diet plans** — Calorie deficit, keto, intermittent fasting (16:8, 18:6, 20:4, OMAD), paleo, mediterranean
- **TDEE calculator** — Mifflin-St Jeor formula with activity level multipliers
- **Water & exercise tracking** — Log intake and workouts with MET-based calorie burn
- **Weight progress charts** — Visualize your weight journey
- **AI health coach** — Daily tips and full chat interface for questions
- **Achievements** — Gamified milestones for streaks and goals
- **PWA** — Install as app on mobile/computer, works offline
- **Dark/light mode** — Your eyes, your choice

## Tech Stack

| Layer | Technology |
|-------|------------|
| Backend | Go 1.22+ (Chi router, sqlc, JWT auth) |
| Frontend | SvelteKit 5 + TailwindCSS v4 + Chart.js |
| Database | PostgreSQL 16 |
| AI | OpenAI GPT-4o / Anthropic Claude Sonnet |
| Container | Docker + docker-compose |

## Quick Start

### Prerequisites

- Docker & Docker Compose installed ([how to install](docs/getting-started.md#prerequisites))
- An OpenAI **or** Anthropic API key ([how to get one](docs/ai-providers.md))

### 1. Clone and Configure

```bash
git clone https://github.com/Sahaj-Tech-ltd/joules.git
cd joules
cp .env.example .env
```

Edit `.env` and fill in your values:

```env
# Required: Choose one AI provider
AI_PROVIDER=openai
OPENAI_API_KEY=sk-...
# Or use Anthropic:
# AI_PROVIDER=anthropic
# ANTHROPIC_API_KEY=sk-ant-...

# Required: Database connection
DATABASE_URL=postgres://joule:joule@db:5432/joule?sslmode=disable

# Required: JWT secret (generate a random string)
JWT_SECRET=your-secure-random-string-here
```

### 2. Run

```bash
docker compose up --build
```

The app will be available at `http://localhost:3000`.

### 3. Create Account

1. Open `http://localhost:3000`
2. Sign up with email/password
3. Check Docker logs for your verification code: `docker compose logs joule`
4. Complete onboarding (height, weight, goals, diet plan)
5. Start tracking!

For a full walkthrough, see the [Getting Started guide](docs/getting-started.md).

## Configuration

| Variable | Required | Description |
|----------|----------|-------------|
| `PORT` | No | App port (default: 3000) |
| `DATABASE_URL` | Yes | PostgreSQL connection string |
| `JWT_SECRET` | Yes | Secret for JWT signing |
| `AI_PROVIDER` | Yes | `openai` or `anthropic` |
| `OPENAI_API_KEY` | Conditional | Required if `AI_PROVIDER=openai` |
| `ANTHROPIC_API_KEY` | Conditional | Required if `AI_PROVIDER=anthropic` |
| `AI_MODEL` | No | Override default model |
| `SMTP_HOST` | No | SMTP server for email verification |
| `SMTP_USER` | No | SMTP username |
| `SMTP_PASS` | No | SMTP password |

Full configuration reference: [docs/configuration.md](docs/configuration.md)

## Architecture

```
docker-compose.yml
├── joule (Go binary + embedded frontend)
│   └── Serves API + static files from single port
└── db (PostgreSQL 16)
    └── Persistent volume for data
```

Single Go binary embeds the compiled SvelteKit frontend. No nginx required.

## Development

### Backend (Go)

```bash
cd backend
go run ./cmd/server          # Run with hot reload
sqlc generate                # Regenerate DB code after schema changes
go test ./...                # Run tests
```

### Frontend (SvelteKit)

```bash
cd frontend
npm install
npm run dev                  # Dev server at localhost:5173
npm run build                # Production build
npm run check                # Type check
```

### Database Migrations

Schema is auto-initialized on startup. To reset:

```bash
docker compose down -v       # Delete volumes
docker compose up --build    # Re-create
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/auth/signup` | Create account |
| `POST` | `/api/auth/verify` | Verify email |
| `POST` | `/api/auth/login` | Login |
| `GET/PUT` | `/api/user/profile` | Profile management |
| `GET` | `/api/user/goals` | Diet goals |
| `POST` | `/api/meals` | Log meal (photo or manual) |
| `GET` | `/api/meals?date=` | Get meals by date |
| `POST` | `/api/water` | Log water intake |
| `POST` | `/api/exercises` | Log exercise |
| `POST` | `/api/weight` | Log weight |
| `GET` | `/api/dashboard/summary` | Daily summary |
| `GET/POST` | `/api/coach/chat` | AI coach chat |
| `GET` | `/api/achievements` | List achievements |
| `GET` | `/api/export/csv?type=` | Export data |

## Project Structure

```
joules/
├── backend/
│   ├── cmd/server/main.go      # Entry point
│   ├── internal/
│   │   ├── auth/               # JWT auth, verification
│   │   ├── user/               # Profile, TDEE calculation
│   │   ├── meal/               # Meal logging, photo AI
│   │   ├── water/              # Water tracking
│   │   ├── exercise/           # Exercise logging, MET calc
│   │   ├── weight/             # Weight history
│   │   ├── coach/              # AI health coach
│   │   ├── achievement/        # Achievement system
│   │   ├── export/             # CSV export
│   │   ├── ai/                 # OpenAI/Anthropic clients
│   │   └── db/                 # Database connection
│   ├── sql/queries/            # sqlc query files
│   └── Dockerfile
├── frontend/
│   ├── src/routes/             # SvelteKit pages
│   ├── src/lib/                # API client, stores
│   ├── src/components/         # UI components
│   └── static/                 # PWA assets
├── docs/                       # Documentation
├── docker-compose.yml
└── README.md
```

## Documentation

| Guide | Description |
|-------|-------------|
| [Getting Started](docs/getting-started.md) | Complete first-time setup — zero experience needed |
| [Configuration](docs/configuration.md) | Every environment variable explained |
| [AI Providers](docs/ai-providers.md) | OpenAI and Anthropic setup, model selection |
| [Email Setup](docs/email-setup.md) | Optional SMTP configuration |
| [Reverse Proxy](docs/reverse-proxy.md) | Expose Joules on the internet with Caddy or Nginx |
| [Usage Guide](docs/usage-guide.md) | How to use the app day-to-day |

## License

MIT License — use it, modify it, share it.

---

Built for people who want control over their health data. No paywalls, no data harvesting, no subscriptions.
