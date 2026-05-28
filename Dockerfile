# Build stage
# Pinned to a specific patch version for reproducible builds.
# Bump deliberately when picking up Go security/bugfix releases.
FROM golang:1.24.13-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod ./

# Download dependencies
RUN go mod download

# Copy source code
COPY *.go ./

# Build arguments for version injection
ARG VERSION=dev
ARG TARGETOS
ARG TARGETARCH

# Build the binary
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w -X main.Version=${VERSION}" \
    -o udp-sender .

# Runtime stage - minimal Alpine image
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 udpsender && \
    adduser -D -u 1000 -G udpsender udpsender

# Copy binary from builder
COPY --from=builder /build/udp-sender /usr/local/bin/udp-sender

# Use non-root user
USER udpsender

WORKDIR /app

# The container requires NET_RAW capability at runtime
# Run with: docker run --cap-add=NET_RAW ...

ENTRYPOINT ["/usr/local/bin/udp-sender"]

