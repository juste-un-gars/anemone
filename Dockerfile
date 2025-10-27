# Anemone - Multi-user NAS with P2P encrypted synchronization
# Copyright (C) 2025 juste-un-gars
# Licensed under the GNU Affero General Public License v3.0

# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

# Copy go mod files
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o anemone ./cmd/anemone

# Runtime stage
FROM alpine:latest

LABEL maintainer="Anemone Project"
LABEL description="Anemone - Multi-user NAS with P2P encrypted synchronization"

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    rclone \
    samba \
    samba-common-tools \
    bash \
    curl

# Create app user
RUN addgroup -g 1000 anemone && \
    adduser -D -u 1000 -G anemone anemone

# Create directories
RUN mkdir -p /app/data/db /app/data/shares /app/data/config && \
    chown -R anemone:anemone /app

# Copy binary from builder
COPY --from=builder /build/anemone /app/anemone

# Copy web assets
COPY --chown=anemone:anemone web /app/web

WORKDIR /app

# Switch to app user
USER anemone

# Expose ports
EXPOSE 8080
EXPOSE 445

# Set environment
ENV ANEMONE_DATA_DIR=/app/data
ENV PORT=8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Run
ENTRYPOINT ["/app/anemone"]
