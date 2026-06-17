# Copyright 2026 Durga Prasad Raju Nadimpalli
# Licensed under the Apache License, Version 2.0
# Build: docker build -f services/collector/Dockerfile .

ARG GO_VERSION=1.22
FROM golang:${GO_VERSION}-alpine AS builder
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY platform/config ./platform/config
COPY services/collector ./services/collector
WORKDIR /src/services/collector
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app ./cmd/collector

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /app /app
USER nobody
EXPOSE 8081
ENTRYPOINT ["/app"]
