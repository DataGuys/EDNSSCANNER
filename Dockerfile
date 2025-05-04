# Build stage
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go module files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o dns-scanner ./cmd/server

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root user
RUN adduser -D -h /app scanner
USER scanner

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder --chown=scanner:scanner /app/dns-scanner .

# Copy static files, templates, and wordlists
COPY --from=builder --chown=scanner:scanner /app/static ./static
COPY --from=builder --chown=scanner:scanner /app/templates ./templates
COPY --from=builder --chown=scanner:scanner /app/wordlists ./wordlists

# Add health check for Azure
HEALTHCHECK --interval=30s --timeout=30s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/ || exit 1

# Expose web interface port
EXPOSE 8080

# Add startup script for Azure compatibility
RUN echo '#!/bin/sh\n\
# Create directories if they don't exist\n\
mkdir -p /app/wordlists /app/static /app/templates\n\
\n\
# Start the application\n\
exec ./dns-scanner "$@"\n\
' > /app/start.sh && chmod +x /app/start.sh

# Start the application
CMD ["/app/start.sh"]
