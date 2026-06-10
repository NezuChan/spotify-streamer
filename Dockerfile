# Build stage
FROM golang:1.26-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev libogg-dev libvorbis-dev flac-dev

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-linkmode external -extldflags "-static"' -o spotify-streamer ./cmd/server

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates libogg libvorbis flac

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/spotify-streamer .
COPY --from=builder /build/config.example.yaml ./config.example.yaml

# Create state directory
RUN mkdir -p /app/data

# Expose port
EXPOSE 8080

# Run as non-root user
RUN addgroup -g 1000 spotify && \
    adduser -D -u 1000 -G spotify spotify && \
    chown -R spotify:spotify /app

USER spotify

ENTRYPOINT ["/app/spotify-streamer"]
CMD ["-config", "/app/config.yaml"]
