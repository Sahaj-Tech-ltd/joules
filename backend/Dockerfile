FROM node:20-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm ci
COPY frontend/ .
RUN npm run build

FROM golang:1.25-alpine AS backend
RUN apk add --no-cache gcc musl-dev
WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
COPY --from=frontend /app/frontend/build/ ./cmd/server/dist/
RUN CGO_ENABLED=0 go build -o /joule ./cmd/server

FROM alpine:3.19
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    # Tesseract OCR — used when OCR_PROVIDER=tesseract
    # Adds ~50 MB to the image. Safe to remove if you don't need local OCR.
    tesseract-ocr \
    tesseract-ocr-data-eng
WORKDIR /app
COPY --from=backend /joule /app/joule
RUN mkdir -p /app/uploads
EXPOSE 3000
CMD ["/app/joule"]
