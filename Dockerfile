# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o hyper2kvm \
    ./cmd/hyper2kvm

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata

# Create non-root user
RUN addgroup -g 1000 hyper2kvm && \
    adduser -D -u 1000 -G hyper2kvm hyper2kvm

# Create directories
RUN mkdir -p /exports && \
    chown -R hyper2kvm:hyper2kvm /exports

# Copy binary from builder
COPY --from=builder /build/hyper2kvm /usr/local/bin/hyper2kvm

# Set user
USER hyper2kvm

# Set working directory
WORKDIR /exports

# Set environment variables
ENV LOG_LEVEL=info \
    DOWNLOAD_WORKERS=3 \
    RETRY_ATTEMPTS=3 \
    PROGRESS_STYLE=bar

# Health check (if needed)
# HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
#     CMD ["/usr/local/bin/hyper2kvm", "--version"] || exit 1

# Run the application
ENTRYPOINT ["/usr/local/bin/hyper2kvm"]
