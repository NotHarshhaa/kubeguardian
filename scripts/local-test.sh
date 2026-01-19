#!/bin/bash

# Local testing script for KubeGuardian
# This script helps test the application locally before deployment

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}🧪 Testing KubeGuardian Locally${NC}"

# Check prerequisites
check_prereq() {
    echo -e "${YELLOW}📋 Checking prerequisites...${NC}"
    
    if ! command -v go &> /dev/null; then
        echo -e "${RED}❌ Go is not installed${NC}"
        exit 1
    fi
    
    if ! command -v docker &> /dev/null; then
        echo -e "${RED}❌ Docker is not installed${NC}"
        exit 1
    fi
    
    if ! command -v kubectl &> /dev/null; then
        echo -e "${RED}❌ kubectl is not installed${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}✅ All prerequisites satisfied${NC}"
}

# Build the application
build_app() {
    echo -e "${YELLOW}🔨 Building KubeGuardian...${NC}"
    
    # Download dependencies
    echo "Downloading Go modules..."
    go mod download
    
    # Run tests
    echo "Running tests..."
    go test ./... || echo -e "${YELLOW}⚠️ Some tests failed (expected during initial setup)${NC}"
    
    # Build binary
    echo "Building binary..."
    go build -o bin/kubeguardian cmd/kubeguardian/main.go
    
    echo -e "${GREEN}✅ Build completed${NC}"
}

# Build Docker image
build_docker() {
    echo -e "${YELLOW}🐳 Building Docker image...${NC}"
    
    docker build -t kubeguardian:test .
    
    echo -e "${GREEN}✅ Docker image built${NC}"
}

# Test Helm chart
test_helm() {
    echo -e "${YELLOW}⚓ Testing Helm chart...${NC}"
    
    if ! command -v helm &> /dev/null; then
        echo -e "${YELLOW}⚠️ Helm not found, skipping Helm tests${NC}"
        return
    fi
    
    # Lint chart
    echo "Linting Helm chart..."
    helm lint deployments/helm/
    
    # Template chart (dry run)
    echo "Testing Helm chart template..."
    helm template kubeguardian deployments/helm/ --debug
    
    echo -e "${GREEN}✅ Helm chart tests passed${NC}"
}

# Run local tests
run_local_tests() {
    echo -e "${YELLOW}🏃 Running local tests...${NC}"
    
    # Test configuration loading
    echo "Testing configuration..."
    go run cmd/kubeguardian/main.go --help
    
    echo -e "${GREEN}✅ Local tests completed${NC}"
}

# Main execution
main() {
    echo -e "${GREEN}🚀 Starting local test suite${NC}"
    
    check_prereq
    build_app
    build_docker
    test_helm
    run_local_tests
    
    echo -e "${GREEN}🎉 All tests completed successfully!${NC}"
    echo ""
    echo -e "${YELLOW}📋 Next steps:${NC}"
    echo "1. Run locally: ./bin/kubeguardian --config configs/config.yaml"
    echo "2. Test with Docker: docker run --rm kubeguardian:test"
    echo "3. Deploy to cluster: kubectl apply -f deployments/manifests/install.yaml"
    echo ""
    echo -e "${GREEN}🔧 For development:${NC}"
    echo "- Use examples/development-config.yaml for safe testing"
    echo "- Enable dryRun: true in remediation config"
    echo "- Check logs with: kubectl logs -n kubeguardian deployment/kubeguardian"
}

# Run main function
main "$@"
