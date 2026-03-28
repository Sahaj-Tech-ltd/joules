# AI Providers

Joules uses AI for two things:
1. **Food photo analysis** — identify food items and estimate calories/macros from a photo
2. **Health coach** — answer questions about nutrition, exercise, and your progress

Both features use the same provider. You configure it once in `.env`.

---

## Supported Providers

### OpenAI

The default. Uses GPT-4o, which has strong vision capabilities for food identification.

```env
AI_PROVIDER=openai
OPENAI_API_KEY=sk-proj-...
```

**Get a key:** [platform.openai.com/api-keys](https://platform.openai.com/api-keys)

**Pricing:** Pay-per-use. Food photo analysis uses vision tokens (more expensive than text). A typical meal photo costs roughly $0.01–0.03.

**Recommended models:**

| Model | Use case |
|-------|----------|
| `gpt-4o` | Best accuracy (default) |
| `gpt-4o-mini` | Faster, ~10× cheaper, slightly less accurate |

### Anthropic Claude

An alternative to OpenAI. Uses Claude Sonnet by default.

```env
AI_PROVIDER=anthropic
ANTHROPIC_API_KEY=sk-ant-...
```

**Get a key:** [console.anthropic.com](https://console.anthropic.com)

**Pricing:** Similar pay-per-use structure to OpenAI.

**Recommended models:**

| Model | Use case |
|-------|----------|
| `claude-sonnet-4-6` | Best accuracy (default) |
| `claude-haiku-4-5-20251001` | Faster, cheaper |

---

## Choosing a Model

Set `AI_MODEL` in `.env` to override the default:

```env
AI_MODEL=gpt-4o-mini
```

**Start with the default.** The defaults (`gpt-4o` and `claude-sonnet-4-6`) give the most accurate food identification. Switch to a cheaper model if you find the costs too high — the difference in everyday use is usually small.

---

## Using a Custom OpenAI-Compatible Endpoint

You can point Joules at any OpenAI-compatible API by setting `OPENAI_BASE_URL`. This lets you use:

- **Local models** via [Ollama](https://ollama.com/) — free, fully private, no API key needed
- **OpenRouter** — access many models through one API key
- **Azure OpenAI** — enterprise deployments

```env
AI_PROVIDER=openai
OPENAI_API_KEY=ollama        # Required by the client but ignored by Ollama
OPENAI_BASE_URL=http://localhost:11434/v1
AI_MODEL=llava               # A vision-capable Ollama model
```

> **Note:** Local vision models are significantly less accurate than GPT-4o for food identification. Results may vary.

---

## What Happens Without an API Key?

Joules won't start if `AI_PROVIDER` is set but the corresponding key is missing. If you see an error like `AI key not configured`, double-check your `.env` and restart:

```bash
docker compose down
docker compose up -d
```
