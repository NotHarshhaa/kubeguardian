.PHONY: build test test-unit test-integration test-benchmark test-security test-chaos test-race coverage clean docker-build docker-push build-arm64 build-optimized help

# Build the binary
build:
	go build -o bin/kubeguardian cmd/kubeguardian/main.go

# Run tests
test: test-unit test-integration test-benchmark test-security

# Unit tests
test-unit:
	@echo "🧪 Running unit tests..."
	go test -v ./pkg/... -short

# Integration tests (requires Kubernetes cluster)
test-integration:
	@echo "🔗 Running integration tests..."
	go test -v ./test/integration/... -tags=integration

# Performance benchmarks
test-benchmark:
	@echo "⚡ Running performance benchmarks..."
	go test -bench=. ./pkg/... -benchmem

# Security tests
test-security:
	@echo "🔒 Running security tests..."
	go test -v ./pkg/config/... -run=TestSecurity

# Chaos engineering tests
test-chaos:
	@echo "🌪️ Running chaos engineering tests..."
	go test -v ./test/chaos/... -tags=chaos

# Race condition tests
test-race:
	@echo "🏃 Running race condition tests..."
	go test -race -short ./pkg/...

# Coverage report
coverage:
	@echo "📊 Generating coverage report..."
	go test -coverprofile=coverage.out ./pkg/...
	go tool cover -html=coverage.out -o coverage.html
	go tool cover -func=coverage.out | grep total

# Memory profiling
memory-profile:
	@echo "🧠 Generating memory profile..."
	go test -memprofile=mem.prof -bench=. ./pkg/...

# CPU profiling
cpu-profile:
	@echo "⚙️ Generating CPU profile..."
	go test -cpuprofile=cpu.prof -bench=. ./pkg/...

# Static analysis
static-analysis:
	@echo "🔍 Running static analysis..."
	go vet ./...
	go fmt ./...
	go mod tidy

# Security scan
security-scan:
	@echo "🔒 Running security scan..."
	govulncheck ./...

# Build tests
build-test:
	@echo "🔨 Testing builds..."
	go build ./cmd/kubeguardian
	go build -race ./cmd/kubeguardian
	rm -f kubeguardian

# Comprehensive test suite
test-all: static-analysis test-unit test-race test-security test-benchmark coverage build-test
	@echo "🎉 Comprehensive testing completed!"

# Quick test (for development)
test-quick: static-analysis test-unit
	@echo "⚡ Quick testing completed!"

# Production readiness check
prod-ready: test-all test-integration security-scan
	@echo "🚀 Production readiness check completed!"

# Clean build artifacts and test files
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html mem.prof cpu.prof kubeguardian
	go clean -testcache

# Docker build (original)
docker-build:
	docker build -t kubeguardian/kubeguardian:latest .

# Docker build optimized for arm64 (fast)
build-arm64:
	docker build --platform=linux/arm64 -t kubeguardian/kubeguardian:arm64 .

# Docker build optimized multi-arch (very fast)
build-optimized:
	docker buildx build --platform linux/amd64,linux/arm64 -t kubeguardian/kubeguardian:latest --push .

# Local arm64 build with buildx
build-arm64-local:
	docker buildx build --platform linux/arm64 -t kubeguardian/kubeguardian:arm64 --load .

# Docker push
docker-push:
	docker push kubeguardian/kubeguardian:latest

# Run locally
run:
	go run cmd/kubeguardian/main.go

# Format code
fmt:
	go fmt ./...

# Vet code
vet:
	go vet ./...

# Generate code
generate:
	go generate ./...

# Setup buildx for multi-arch builds
setup-buildx:
	docker buildx create --name multiarch --use
	docker buildx inspect --bootstrap
