# ---------------------------------------------------------
# STAGE 1: Builder
# ---------------------------------------------------------
FROM golang:1.26.1-alpine3.23 AS builder

# Required for CGO builds (SQLite in profiler engine)
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Copy dependency files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# 1. Build the Crypto Profiler engine/server
# Uses CGO because SQLite support requires it
RUN CGO_ENABLED=1 GOOS=linux go build -o profiler ./cmd/profiler

# 2. Build the validator/client
# Keep this only if you still want a separate validator binary
RUN CGO_ENABLED=0 GOOS=linux go build -o validator ./main.go

# ---------------------------------------------------------
# STAGE 2: Runtime
# ---------------------------------------------------------
FROM alpine:3.23

# Install certs for outbound HTTPS/API calls
RUN apk add --no-cache ca-certificates

WORKDIR /root/

# Copy binaries
COPY --from=builder /app/profiler ./profiler
COPY --from=builder /app/validator ./validator

# Entrypoint routing:
# - "server" runs the profiler engine
# - anything else is passed to the validator
RUN echo '#!/bin/sh' > /entrypoint.sh && \
    echo 'if [ "$1" = "server" ]; then' >> /entrypoint.sh && \
    echo '    shift' >> /entrypoint.sh && \
    echo '    exec ./profiler "$@"' >> /entrypoint.sh && \
    echo 'else' >> /entrypoint.sh && \
    echo '    exec ./validator "$@"' >> /entrypoint.sh && \
    echo 'fi' >> /entrypoint.sh && \
    chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]