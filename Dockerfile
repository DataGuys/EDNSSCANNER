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
RUN apk add --no-cache ca-certificates tzdata

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

# Expose web interface port
EXPOSE 8080

# Start the application
CMD ["./dns-scanner"]