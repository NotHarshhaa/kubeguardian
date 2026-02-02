#!/bin/bash

# Comprehensive Test Runner for KubeGuardian
# This script runs all types of tests to identify potential issues

set -e

echo "🚀 Starting KubeGuardian Comprehensive Testing Suite"
echo "=================================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Function to run a test category
run_test_category() {
    local category=$1
    local description=$2
    local command=$3
    
    echo -e "\n${BLUE}📋 Running ${category} Tests${NC}"
    echo "${description}"
    echo "----------------------------------------"
    
    if eval "$command"; then
        echo -e "${GREEN}✅ ${category} tests passed${NC}"
        ((PASSED_TESTS++))
    else
        echo -e "${RED}❌ ${category} tests failed${NC}"
        ((FAILED_TESTS++))
    fi
    ((TOTAL_TESTS++))
}

# Function to run tests with coverage
run_with_coverage() {
    local category=$1
    local command=$2
    
    echo -e "\n${BLUE}📊 Running ${category} with Coverage${NC}"
    echo "----------------------------------------"
    
    if eval "$command"; then
        echo -e "${GREEN}✅ ${category} coverage generated${NC}"
    else
        echo -e "${RED}❌ ${category} coverage failed${NC}"
    fi
}

# 1. Static Analysis and Code Quality
echo -e "\n${YELLOW}🔍 Static Analysis and Code Quality${NC}"
echo "=========================================="

# Go vet
echo "Running go vet..."
if go vet ./...; then
    echo -e "${GREEN}✅ go vet passed${NC}"
else
    echo -e "${RED}❌ go vet failed${NC}"
    ((FAILED_TESTS++))
fi

# Go fmt
echo "Running go fmt..."
if [ "$(gofmt -l . | wc -l)" -eq 0 ]; then
    echo -e "${GREEN}✅ go fmt passed${NC}"
else
    echo -e "${RED}❌ go fmt found issues${NC}"
    gofmt -l .
    ((FAILED_TESTS++))
fi

# Go mod tidy
echo "Running go mod tidy..."
if go mod tidy; then
    echo -e "${GREEN}✅ go mod tidy passed${NC}"
else
    echo -e "${RED}❌ go mod tidy failed${NC}"
    ((FAILED_TESTS++))
fi

# 2. Unit Tests
run_test_category "Unit" "Basic functionality tests" "go test -v ./pkg/... -short"

# 3. Integration Tests
if command -v kubectl &> /dev/null && kubectl cluster-info &> /dev/null; then
    run_test_category "Integration" "Kubernetes cluster integration" "go test -v ./test/integration/... -tags=integration"
else
    echo -e "\n${YELLOW}⚠️  Skipping integration tests - no Kubernetes cluster available${NC}"
fi

# 4. Performance Benchmarks
run_test_category "Benchmark" "Performance and load tests" "go test -bench=. ./pkg/... -benchmem"

# 5. Security Tests
run_test_category "Security" "Security validation and input sanitization" "go test -v ./pkg/config/... -run=TestSecurity"

# 6. Chaos Engineering Tests (if available)
if [ -d "test/chaos" ]; then
    run_test_category "Chaos" "Chaos engineering and resilience tests" "go test -v ./test/chaos/... -tags=chaos"
else
    echo -e "\n${YELLOW}⚠️  Chaos engineering tests not available${NC}"
fi

# 7. Race Condition Tests
run_test_category "Race" "Race condition detection" "go test -race -short ./pkg/..."

# 8. Coverage Report
echo -e "\n${BLUE}📈 Generating Coverage Report${NC}"
echo "=========================================="

# Generate coverage for main packages
if go test -coverprofile=coverage.out ./pkg/...; then
    echo -e "${GREEN}✅ Coverage report generated${NC}"
    
    # Generate HTML coverage report
    if go tool cover -html=coverage.out -o coverage.html; then
        echo -e "${GREEN}✅ HTML coverage report generated: coverage.html${NC}"
    fi
    
    # Show coverage percentage
    COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
    echo -e "${BLUE}📊 Total Coverage: ${COVERAGE}${NC}"
else
    echo -e "${RED}❌ Coverage report failed${NC}"
fi

# 9. Memory and Performance Analysis
echo -e "\n${BLUE}🧠 Memory and Performance Analysis${NC}"
echo "=========================================="

# Test memory usage
echo "Running memory profiling tests..."
if go test -memprofile=mem.prof -bench=. ./pkg/... 2>/dev/null; then
    echo -e "${GREEN}✅ Memory profile generated${NC}"
else
    echo -e "${YELLOW}⚠️  Memory profiling failed or not available${NC}"
fi

# Test CPU profiling
echo "Running CPU profiling tests..."
if go test -cpuprofile=cpu.prof -bench=. ./pkg/... 2>/dev/null; then
    echo -e "${GREEN}✅ CPU profile generated${NC}"
else
    echo -e "${YELLOW}⚠️  CPU profiling failed or not available${NC}"
fi

# 10. Dependency Security Scan
echo -e "\n${BLUE}🔒 Dependency Security Scan${NC}"
echo "=========================================="

if command -v govulncheck &> /dev/null; then
    echo "Running govulncheck..."
    if govulncheck ./...; then
        echo -e "${GREEN}✅ No known vulnerabilities found${NC}"
    else
        echo -e "${YELLOW}⚠️  Vulnerabilities found or scan failed${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  govulncheck not available, install with: go install golang.org/x/vuln/cmd/govulncheck@latest${NC}"
fi

# 11. Build Tests
echo -e "\n${BLUE}🔨 Build Tests${NC}"
echo "=========================================="

# Test normal build
echo "Testing normal build..."
if go build ./cmd/kubeguardian; then
    echo -e "${GREEN}✅ Build successful${NC}"
    rm -f kubeguardian  # Clean up
else
    echo -e "${RED}❌ Build failed${NC}"
    ((FAILED_TESTS++))
fi

# Test race build
echo "Testing race build..."
if go build -race ./cmd/kubeguardian; then
    echo -e "${GREEN}✅ Race build successful${NC}"
    rm -f kubeguardian  # Clean up
else
    echo -e "${RED}❌ Race build failed${NC}"
    ((FAILED_TESTS++))
fi

# 12. Final Summary
echo -e "\n${BLUE}📋 Test Summary${NC}"
echo "=========================================="
echo -e "Total Test Categories: ${TOTAL_TESTS}"
echo -e "${GREEN}Passed: ${PASSED_TESTS}${NC}"
echo -e "${RED}Failed: ${FAILED_TESTS}${NC}"

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "\n${GREEN}🎉 All tests passed! KubeGuardian is ready for production.${NC}"
    exit 0
else
    echo -e "\n${RED}❌ Some tests failed. Please review and fix the issues.${NC}"
    exit 1
fi
