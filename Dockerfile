# Stage 1: Build the binary
FROM golang:1.23-alpine AS builder

WORKDIR /app

# 1. Copy dependencies first (caching layer)
COPY go.mod ./
# COPY go.sum ./   <-- Uncomment this if you have a go.sum file

# 2. Copy THE ENTIRE PROJECT (including internal/ folder)
COPY . .

# 3. Build the application
# We use -o validator to match the entrypoint name
RUN CGO_ENABLED=0 GOOS=linux go build -o validator main.go

# Stage 2: Create the runtime image
FROM alpine:latest

# Install SSL certs for HTTPS connections
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/validator .

# Run it
ENTRYPOINT ["./validator"]