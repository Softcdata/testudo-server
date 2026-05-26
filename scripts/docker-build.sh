#!/bin/bash
# Docker build script for disaster-server with multi-architecture support
# Usage:
#   ./scripts/docker-build.sh                          # Build for host architecture
#   ./scripts/docker-build.sh --platform linux/arm64   # Build for ARM64
#   ./scripts/docker-build.sh --buildx --push          # Build and push multi-arch image

set -e

# Default values
IMAGE_NAME="${IMAGE_NAME:-disaster-server}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
VERSION="${VERSION:-dev}"
PLATFORMS="${PLATFORMS:-linux/amd64,linux/arm64}"
OPERATOR_CONTEXT="${OPERATOR_CONTEXT:-../disaster-operator}"
BUILDX_BUILDER="disaster-server-builder"

# Parse arguments
USE_BUILDX=false
PUSH=false
PLATFORM=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --buildx)
            USE_BUILDX=true
            shift
            ;;
        --push)
            PUSH=true
            shift
            ;;
        --platform)
            PLATFORM="$2"
            shift 2
            ;;
        --tag|-t)
            IMAGE_TAG="$2"
            shift 2
            ;;
        --image|-i)
            IMAGE_NAME="$2"
            shift 2
            ;;
        --version|-v)
            VERSION="$2"
            shift 2
            ;;
        --platforms)
            PLATFORMS="$2"
            shift 2
            ;;
        --help|-h)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --buildx              Use docker buildx for multi-arch build"
            echo "  --push                Push image after build (requires --buildx)"
            echo "  --platform <platform> Target platform (e.g., linux/arm64)"
            echo "  --platforms <list>    Comma-separated platforms for buildx"
            echo "  --tag, -t <tag>       Image tag (default: latest)"
            echo "  --image, -i <name>    Image name (default: disaster-server)"
            echo "  --version, -v <ver>   Version to embed in binary (default: dev)"
            echo "  OPERATOR_CONTEXT      Path to disaster-operator checkout (default: ../disaster-operator)"
            echo "  --help, -h            Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0                                    # Build for host arch"
            echo "  $0 --platform linux/arm64             # Build for ARM64"
            echo "  $0 --buildx --push --tag v1.0.0       # Multi-arch build and push"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

FULL_IMAGE="${IMAGE_NAME}:${IMAGE_TAG}"

if [ ! -f "${OPERATOR_CONTEXT}/go.mod" ]; then
    echo "Error: operator context not found at ${OPERATOR_CONTEXT}; expected ${OPERATOR_CONTEXT}/go.mod"
    exit 1
fi

if [ "$USE_BUILDX" = true ]; then
    echo "Building multi-arch image: ${FULL_IMAGE}"
    echo "Platforms: ${PLATFORMS}"
    

    # Create builder if not exists
    docker buildx create --name ${BUILDX_BUILDER} 2>/dev/null || true
    docker buildx use ${BUILDX_BUILDER}
    
    BUILDX_ARGS="--platform ${PLATFORMS} --build-context operator_source=${OPERATOR_CONTEXT} --tag ${FULL_IMAGE} --build-arg VERSION=${VERSION}"
    
    if [ "$PUSH" = true ]; then
        BUILDX_ARGS="${BUILDX_ARGS} --push"
    else
        # For local multi-arch build without push, use --load (only works for single platform)
        echo "Warning: --buildx without --push will only build, not load to local docker"
    fi
    
    docker buildx build ${BUILDX_ARGS}  .
    
    # Cleanup builder
    docker buildx rm ${BUILDX_BUILDER} 2>/dev/null || true
else
    # Standard docker build
    BUILD_ARGS="--build-arg VERSION=${VERSION}"
    
    if [ -n "$PLATFORM" ]; then
        echo "Building for platform: ${PLATFORM}"
        BUILD_ARGS="${BUILD_ARGS} --platform ${PLATFORM}"
    else
        echo "Building for host architecture"
    fi
    
    docker build ${BUILD_ARGS} --build-context operator_source=${OPERATOR_CONTEXT} -t ${FULL_IMAGE} --no-cache --provenance=false .

    # Push to SWR
    # docker tag disaster-server:latest registry.example.com/disaster/disaster-server:latest
    # docker push registry.example.com/disaster/disaster-server:latest
fi

echo ""
echo "Build complete: ${FULL_IMAGE}"
