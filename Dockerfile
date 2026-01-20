# ---------------------------------------------------------
# STAGE 1: The Builder (Builds EVERYTHING)
# ---------------------------------------------------------
FROM golang:1.23-alpine AS builder

# Install C-compiler tools (Required for the Engine's SQLite)
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# 1. Build the WATCHLIST ENGINE (Server)
# Requires CGO_ENABLED=1 because it uses SQLite
RUN CGO_ENABLED=1 GOOS=linux go build -o engine cmd/engine/main.go

# 2. Build the VALIDATOR (Client)
# Uses CGO_ENABLED=0 for a static, lightweight binary
RUN CGO_ENABLED=0 GOOS=linux go build -o validator main.go

# ---------------------------------------------------------
# STAGE 2: The Runtime (Universal Image)
# ---------------------------------------------------------
FROM alpine:latest

# Install certificates for HTTPS requests
RUN apk add --no-cache ca-certificates

WORKDIR /root/

# Copy BOTH binaries from the builder
COPY --from=builder /app/engine ./engine
COPY --from=builder /app/validator ./validator

# Create a simple entrypoint script to route commands
# If the user types "server", we run the engine.
# Otherwise, we pass the arguments to the validator.
RUN echo '#!/bin/sh' > /entrypoint.sh && \
    echo 'if [ "$1" = "server" ]; then' >> /entrypoint.sh && \
    echo '    exec ./engine' >> /entrypoint.sh && \
    echo 'else' >> /entrypoint.sh && \
    echo '    exec ./validator "$@"' >> /entrypoint.sh && \
    echo 'fi' >> /entrypoint.sh && \
    chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]