# OCR — Tesseract Local Image Processing

By default, food photo analysis sends the full image to your AI provider (GPT-4o / Claude) which reads any visible text as part of its vision analysis. This is the simplest setup.

If you set `OCR_PROVIDER=tesseract`, Joules instead uses **Tesseract OCR** — an open-source text recognition engine bundled in the Docker image — to extract text from photos first, then passes only the extracted text to the AI for parsing.

## When to use Tesseract

| Scenario | Recommendation |
|----------|---------------|
| Most food photos (plates, fresh food) | Default AI vision — more accurate |
| Nutrition labels, product packaging | Tesseract — cheaper, often more accurate |
| Restaurant menus, receipts | Tesseract — reads text reliably |
| Mixed (both food and labels) | Either; Tesseract falls back to AI vision if it extracts little text |

## Setup

Add to your `.env`:

```env
OCR_PROVIDER=tesseract
```

Tesseract is pre-installed in the Docker image — no extra steps needed.

## How it works

1. Photo comes in from the user
2. Image is converted to grayscale (improves OCR accuracy)
3. Tesseract extracts text from the image
4. If the extracted text is meaningful (>20 characters), it's sent to the AI text model for parsing
5. If little/no text is extracted, falls back to sending the full image to the AI vision model

The fallback means Tesseract mode is safe for all photo types — non-label photos just fall through to normal AI vision.

## Vision token savings

When Tesseract successfully extracts text:
- **OpenAI**: uses the text model instead of vision → ~10× cheaper per photo
- **Anthropic**: same — text input instead of image input

When Tesseract falls back to vision, costs are identical to the default.

## Confidence scores

- Items identified from Tesseract-extracted label text: `confidence: 0.95+`
- Items estimated from extracted text descriptions: `confidence: 0.6-0.8`
- Items identified from AI vision (fallback): same as default behavior

## Image size

Tesseract works best on reasonably sized images. Very small images (< 100×100 px) or very blurry images may return poor results and fall back to AI vision automatically.
