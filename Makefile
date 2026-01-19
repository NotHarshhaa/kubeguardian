.PHONY: build test clean docker-build docker-push build-arm64 build-optimized

# Build the binary
build:
	go build -o bin/kubeguardian cmd/kubeguardian/main.go

# Run tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -rf bin/

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
