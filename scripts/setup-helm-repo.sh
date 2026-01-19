#!/bin/bash

# Setup script for KubeGuardian Helm repository
# This script helps set up the GitHub Pages-based Helm chart repository

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
REPO_OWNER="NotHarshhaa"
REPO_NAME="kubeguardian"
CHART_PATH="deployments/helm"
PAGES_BRANCH="gh-pages"

echo -e "${GREEN}🚀 Setting up KubeGuardian Helm Repository${NC}"

# Check if gh CLI is installed
if ! command -v gh &> /dev/null; then
    echo -e "${RED}❌ GitHub CLI (gh) is not installed. Please install it first:${NC}"
    echo "https://cli.github.com/manual/installation"
    exit 1
fi

# Check if we're in the right directory
if [ ! -d "$CHART_PATH" ]; then
    echo -e "${RED}❌ Chart directory not found: $CHART_PATH${NC}"
    echo "Please run this script from the repository root"
    exit 1
fi

# Check if Helm is installed
if ! command -v helm &> /dev/null; then
    echo -e "${RED}❌ Helm is not installed. Please install it first:${NC}"
    echo "https://helm.sh/docs/intro/install/"
    exit 1
fi

echo -e "${YELLOW}📋 Repository Information:${NC}"
echo "Owner: $REPO_OWNER"
echo "Repository: $REPO_NAME"
echo "Chart Path: $CHART_PATH"

# Create gh-pages branch if it doesn't exist
echo -e "${YELLOW}🌳 Setting up gh-pages branch...${NC}"
if ! git show-ref --verify --quiet refs/heads/$PAGES_BRANCH; then
    echo "Creating $PAGES_BRANCH branch..."
    git checkout --orphan $PAGES_BRANCH
    git rm -rf .
    echo "# KubeGuardian Helm Charts" > README.md
    echo "Helm charts for KubeGuardian" >> README.md
    git add README.md
    git commit -m "Initial gh-pages setup"
    git checkout master
else
    echo "$PAGES_BRANCH branch already exists"
fi

# Install chart-releaser
echo -e "${YELLOW}📦 Installing chart-releaser...${NC}"
CR_VERSION="v1.5.0"
if ! command -v cr &> /dev/null; then
    echo "Installing chart-releaser $CR_VERSION..."
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        wget https://github.com/helm/chart-releaser/releases/download/$CR_VERSION/chart-releaser_${CR_VERSION}_darwin_amd64.tar.gz
        tar -xvf chart-releaser_${CR_VERSION}_darwin_amd64.tar.gz
        sudo mv cr /usr/local/bin/cr
    else
        # Linux
        wget https://github.com/helm/chart-releaser/releases/download/$CR_VERSION/chart-releaser_${CR_VERSION}_linux_amd64.tar.gz
        tar -xvf chart-releaser_${CR_VERSION}_linux_amd64.tar.gz
        sudo mv cr /usr/local/bin/cr
    fi
    rm chart-releaser_${CR_VERSION}_*.tar.gz
else
    echo "chart-releaser already installed"
fi

# Lint the chart
echo -e "${YELLOW}🔍 Linting Helm chart...${NC}"
helm lint $CHART_PATH/

# Package the chart
echo -e "${YELLOW}📦 Packaging Helm chart...${NC}"
mkdir -p .cr-release-packages
helm package $CHART_PATH/ --destination .cr-release-packages

# Upload and index the chart
echo -e "${YELLOW}⬆️ Uploading and indexing chart...${NC}"
cr upload -o $REPO_OWNER -r $REPO_NAME
cr index -o $REPO_OWNER -r $REPO_NAME --push

# Enable GitHub Pages
echo -e "${YELLOW}🌐 Enabling GitHub Pages...${NC}"
gh api repos/:owner/:repo/pages -X POST -f source[branch]=gh-pages -f source[path]=/ || echo "Pages might already be enabled"

# Get the Pages URL
PAGES_URL=$(gh api repos/:owner/:repo/pages --jq '.html_url')
echo -e "${GREEN}✅ GitHub Pages enabled at: $PAGES_URL${NC}"

# Test the repository
echo -e "${YELLOW}🧪 Testing Helm repository...${NC}"
helm repo add kubeguardian $PAGES_URL
helm repo update
helm search repo kubeguardian

echo -e "${GREEN}🎉 Helm repository setup complete!${NC}"
echo -e "${GREEN}📋 Next steps:${NC}"
echo "1. Push changes to GitHub: git push origin master"
echo "2. Push gh-pages branch: git push origin gh-pages"
echo "3. Users can now install with: helm install kubeguardian kubeguardian/kubeguardian"
echo ""
echo -e "${YELLOW}📝 Repository URL: $PAGES_URL${NC}"
echo -e "${YELLOW}📦 Chart package location: .cr-release-packages/${NC}"
