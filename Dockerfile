# ── dev ────────────────────────────────────────────────────────────────────────
FROM golang:1.26.2-alpine AS dev

RUN go install github.com/air-verse/air@latest

WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download

CMD ["air", "-c", ".air.toml"]

# ── builder ────────────────────────────────────────────────────────────────────
FROM golang:1.26.2-alpine AS builder

WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/server ./cmd/api

# ── prod ───────────────────────────────────────────────────────────────────────
FROM scratch AS prod

COPY --from=builder /app/server /server

EXPOSE 8080
ENTRYPOINT ["/server"]
