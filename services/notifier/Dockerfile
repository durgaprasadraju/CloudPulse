# Copyright 2026 Durga Prasad Raju Nadimpalli
# Licensed under the Apache License, Version 2.0

ARG GO_VERSION=1.22
FROM golang:${GO_VERSION}-alpine AS builder
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY platform/config ./platform/config
COPY services/notifier ./services/notifier
WORKDIR /src/services/notifier
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app ./cmd/notifier

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /app /app
USER nobody
EXPOSE 8084
ENTRYPOINT ["/app"]
