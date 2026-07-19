# Multi-stage Dockerfile for CloudPulse Go microservices.
# Build a specific service with:
#   docker build --build-arg SERVICE=api-gateway -t cloudpulse/api-gateway .
# Supported SERVICE values: api-gateway | trip-service | driver-service | payment-service

ARG GO_VERSION=1.23
ARG SERVICE=api-gateway

# -----------------------------------------------------------------------------
# Stage 1: build
# -----------------------------------------------------------------------------
FROM golang:${GO_VERSION}-alpine AS builder

ARG SERVICE
WORKDIR /src

RUN apk add --no-cache git ca-certificates

# Cache module downloads
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Resolve build path / binary name per service
RUN set -eux; \
    case "${SERVICE}" in \
      api-gateway) \
        PKG=./services/api-gateway; BIN=api-gateway ;; \
      trip-service) \
        PKG=./services/trip-service/cmd; BIN=trip-service ;; \
      driver-service) \
        PKG=./services/driver-service; BIN=driver-service ;; \
      payment-service) \
        PKG=./services/payment-service/cmd; BIN=payment-service ;; \
      *) \
        echo "Unknown SERVICE=${SERVICE}" >&2; exit 1 ;; \
    esac; \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
      go build -ldflags="-s -w" -o /out/${BIN} ${PKG}

# -----------------------------------------------------------------------------
# Stage 2: runtime
# -----------------------------------------------------------------------------
FROM alpine:3.20 AS runtime

ARG SERVICE
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -H -u 10001 appuser

# Copy the binary built above (name matches SERVICE for api/driver;
# trip/payment use the same SERVICE string as the binary name).
COPY --from=builder /out/${SERVICE} /app/service

USER appuser

# api-gateway listens on HTTP_ADDR (default :8081; compose overrides to :8080)
EXPOSE 8080 8081 9092 9093 9004

ENTRYPOINT ["/app/service"]
