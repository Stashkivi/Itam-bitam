FROM golang:1.22-alpine AS builder

WORKDIR /src
RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/server ./cmd/server

# ── Runtime ─────────────────────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12

COPY --from=builder /out/server /server
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 8443
ENTRYPOINT ["/server"]
