#!/bin/bash

# Build and push script for Trafiks Docker images
# Usage: ./build-and-push.sh [VERSION] [DOCKERHUB_USERNAME]
# Example: ./build-and-push.sh 1.16.0 myusername

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

VERSION=${1:-${VERSION:-"latest"}}
DOCKERHUB_USERNAME=${2:-${DOCKERHUB_USERNAME:-""}}

if [ -z "$DOCKERHUB_USERNAME" ]; then
    echo -e "${RED}Error: DockerHub username is required${NC}"
    echo "Usage: $0 [VERSION] [DOCKERHUB_USERNAME]"
    echo "   or: VERSION=1.16.0 DOCKERHUB_USERNAME=myusername $0"
    echo "Example: $0 1.16.0 myusername"
    exit 1
fi

BACKEND_IMAGE="${DOCKERHUB_USERNAME}/trafiks-backend"
AGENT_IMAGE="${DOCKERHUB_USERNAME}/trafiks-agent"

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Trafiks Docker Build and Push${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Version: ${VERSION}"
echo "DockerHub Username: ${DOCKERHUB_USERNAME}"
echo "Backend Image: ${BACKEND_IMAGE}:${VERSION}"
echo "Agent Image: ${AGENT_IMAGE}:${VERSION}"
echo ""

if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}Error: Docker is not running${NC}"
    exit 1
fi

if ! docker info | grep -q "Username"; then
    echo -e "${YELLOW}Warning: Not logged in to DockerHub${NC}"
    echo "Please run: docker login"
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

echo -e "${GREEN}Building UI...${NC}"
cd web/trafiks-ui
if [ ! -d "node_modules" ]; then
    echo "Installing npm dependencies..."
    npm install
fi
npm run build
cd "$SCRIPT_DIR"
echo -e "${GREEN}UI build complete!${NC}"
echo ""

echo -e "${GREEN}Building backend image...${NC}"
docker build \
    -f Dockerfile.backend \
    -t "${BACKEND_IMAGE}:${VERSION}" \
    -t "${BACKEND_IMAGE}:latest" \
    .
echo -e "${GREEN}Backend image built successfully!${NC}"
echo ""

echo -e "${GREEN}Building agent image...${NC}"
docker build \
    -f Dockerfile.agent \
    -t "${AGENT_IMAGE}:${VERSION}" \
    -t "${AGENT_IMAGE}:latest" \
    .
echo -e "${GREEN}Agent image built successfully!${NC}"
echo ""

if [ -z "$CI" ] && [ -z "$GITHUB_ACTIONS" ]; then
    echo -e "${YELLOW}Ready to push images to DockerHub${NC}"
    echo "Backend: ${BACKEND_IMAGE}:${VERSION} and ${BACKEND_IMAGE}:latest"
    echo "Agent: ${AGENT_IMAGE}:${VERSION} and ${AGENT_IMAGE}:latest"
    read -p "Push to DockerHub? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${YELLOW}Build complete. Images not pushed.${NC}"
        exit 0
    fi
else
    echo -e "${GREEN}CI/CD mode detected. Pushing images automatically...${NC}"
fi

echo -e "${GREEN}Pushing backend images...${NC}"
docker push "${BACKEND_IMAGE}:${VERSION}"
docker push "${BACKEND_IMAGE}:latest"
echo -e "${GREEN}Backend images pushed successfully!${NC}"
echo ""

echo -e "${GREEN}Pushing agent images...${NC}"
docker push "${AGENT_IMAGE}:${VERSION}"
docker push "${AGENT_IMAGE}:latest"
echo -e "${GREEN}Agent images pushed successfully!${NC}"
echo ""

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}All images built and pushed successfully!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Backend: ${BACKEND_IMAGE}:${VERSION}"
echo "Agent: ${AGENT_IMAGE}:${VERSION}"
echo ""
echo "You can now use these images in your Helm chart:"
echo "  backend.image.repository: ${DOCKERHUB_USERNAME}/trafiks-backend"
echo "  backend.image.tag: ${VERSION}"
echo "  agent.image.repository: ${DOCKERHUB_USERNAME}/trafiks-agent"
echo "  agent.image.tag: ${VERSION}"
