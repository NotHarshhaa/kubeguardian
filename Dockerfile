# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s' \
    -a \
    -o kubeguardian \
    cmd/kubeguardian/main.go

# Final stage
FROM gcr.io/distroless/static:nonroot

# Import the user and group files from the builder
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Copy the CA certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Copy the binary
COPY --from=builder /app/kubeguardian /kubeguardian

# Use non-root user
USER 65532:65532

# Expose ports
EXPOSE 8080 8081

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD ["/kubeguardian", "--health-probe-bind-address=0.0.0.0:8081"] || exit 1

# Set the entrypoint
ENTRYPOINT ["/kubeguardian"]

# Default arguments
CMD ["--metrics-bind-address=0.0.0.0:8080", "--health-probe-bind-address=0.0.0.0:8081"]
