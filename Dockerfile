# Dockerfile (Alpine variant) - Default multi-arch image
#
# NOTE: This is the default Dockerfile. For optimized variants, use:
#   - Dockerfile.alpine (full-featured, recommended for development)
#   - Dockerfile.distroless (minimal, recommended for production)
#   - Dockerfile.scratch (smallest, for advanced use cases)
#
# Build the binary first with: task build-linux-amd64
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates git

WORKDIR /

# Copy the pre-built binary
# Default to amd64, override with --build-arg ARCH=arm64 for ARM
ARG ARCH=amd64
COPY build/goblet-server-linux-${ARCH} /goblet-server

# Ensure binary is executable
RUN chmod +x /goblet-server

# Create cache directory
RUN mkdir -p /cache

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1

EXPOSE 8080

ENTRYPOINT ["/goblet-server"]
