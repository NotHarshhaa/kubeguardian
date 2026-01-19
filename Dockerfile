# Simple multi-stage build for KubeGuardian
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files first (for better caching)
COPY go.mod go.sum ./

# Download dependencies (cached if go.mod doesn't change)
RUN go mod download && go mod verify

# Copy only necessary source files
COPY cmd/ ./cmd/
COPY pkg/ ./pkg/

# Ensure dependencies are available and build the application
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o kubeguardian ./cmd/kubeguardian

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/kubeguardian .

# Change ownership
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose ports
EXPOSE 8080 8081

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD ["./kubeguardian", "--help"] || exit 1

# Run the application
ENTRYPOINT ["./kubeguardian"]
CMD ["--metrics-bind-address=0.0.0.0:8080", "--health-probe-bind-address=0.0.0.0:8081"]
