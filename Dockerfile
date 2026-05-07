# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /workspace

# Install build dependencies
RUN apk add --no-cache gcc musl-dev sqlite-dev

# Copy source
COPY . .

# Build binary
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w -X main.Version=1.0.0" \
    -o junkyard-server ./cmd/junkyard-server && \
    go build -ldflags="-s -w -X main.Version=1.0.0" \
    -o junk ./cmd/junkyard-cli

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 junkyard && \
    adduser -D -u 1000 -G junkyard junkyard

# Create data directory
RUN mkdir -p /var/lib/junkyard && \
    chown -R junkyard:junkyard /var/lib/junkyard

# Copy binaries from builder
COPY --from=builder /workspace/junkyard-server /usr/local/bin/
COPY --from=builder /workspace/junk /usr/local/bin/

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

# Switch to non-root user
USER junkyard

# Expose ports
EXPOSE 8080 5514

# Default command
CMD ["junkyard-server", \
     "--http-addr", "0.0.0.0:8080", \
     "--syslog-addr", "0.0.0.0:5514", \
     "--db-path", "/var/lib/junkyard/logs.db", \
     "--retention-days", "14"]
