# ── Build stage ──────────────────────────────────────────────────────────────
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /cinema-server ./cmd/server

# ── Runtime stage ─────────────────────────────────────────────────────────────
FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /cinema-server .
EXPOSE 8080
CMD ["./cinema-server"]