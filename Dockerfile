# ── Build stage ───────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

# git is needed for version stamping; build-base is not required because the
# server binary is pure Go (CGO is only used by the desktop/Wails build).
RUN apk add --no-cache git

WORKDIR /src

# Cache dependencies first for faster rebuilds.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Stamp version info into the tft package (same vars /health reports).
ARG VERSION=docker
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown
ENV CGO_ENABLED=0
RUN go build \
    -ldflags "-s -w \
      -X 'github.com/sagerlabs/awesome/tft.Version=${VERSION}' \
      -X 'github.com/sagerlabs/awesome/tft.GitCommit=${GIT_COMMIT}' \
      -X 'github.com/sagerlabs/awesome/tft.BuildTime=${BUILD_TIME}'" \
    -o /out/tft-copilot ./main.go

# ── Runtime stage ─────────────────────────────────────────────────────────────
FROM alpine:3.20

# ca-certificates: outbound HTTPS to the LLM provider.
# wget (busybox): used by the container healthcheck.
RUN apk add --no-cache ca-certificates && \
    adduser -D -u 10001 appuser

WORKDIR /app

# Binary plus the data the server reads at runtime. The knowledge store path
# is relative ("tft/knowledge/data"), so the layout under /app must match the repo.
COPY --from=builder /out/tft-copilot /app/tft-copilot
COPY --from=builder /src/metadata/tft-meta/data /app/metadata/tft-meta/data
COPY --from=builder /src/tft/knowledge/data /app/tft/knowledge/data

USER appuser

ENV PORT=8080
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -qO- "http://127.0.0.1:${PORT}/v1/tft/health" || exit 1

ENTRYPOINT ["/app/tft-copilot"]
