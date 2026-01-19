.PHONY: build test clean docker-build docker-push

# Build the binary
build:
	go build -o bin/kubeguardian cmd/kubeguardian/main.go

# Run tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -rf bin/

# Docker build
docker-build:
	docker build -t kubeguardian/kubeguardian:latest .

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
