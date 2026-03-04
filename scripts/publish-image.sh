#!/bin/bash
set -e

# Default values
IMAGE_REPO=${IMAGE_REPO:-"ntb/gpusprint"}
IMAGE_TAG=${IMAGE_TAG:-"latest"}
PLATFORMS=${PLATFORMS:-"linux/amd64,linux/arm64"}

echo "Building and pushing $IMAGE_REPO:$IMAGE_TAG for platforms $PLATFORMS..."

# Ensure we're in the project root
cd "$(dirname "$0")/.."

# Check if docker buildx is available
if ! docker buildx version > /dev/null 2>&1; then
    echo "Error: docker buildx is not available. Please install it to build multi-platform images."
    exit 1
fi

# Create a new builder instance if needed
BUILDER_NAME="gpusprint-builder"
if ! docker buildx inspect $BUILDER_NAME > /dev/null 2>&1; then
    echo "Creating new buildx builder instance: $BUILDER_NAME"
    docker buildx create --name $BUILDER_NAME --use
else
    echo "Using existing buildx builder instance: $BUILDER_NAME"
    docker buildx use $BUILDER_NAME
fi

# Build and push the image
docker buildx build \
    --platform "$PLATFORMS" \
    --tag "$IMAGE_REPO:$IMAGE_TAG" \
    --push \
    .

echo "Successfully built and pushed $IMAGE_REPO:$IMAGE_TAG"
