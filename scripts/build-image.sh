#!/usr/bin/env bash
set -euo pipefail

IMAGE_NAME="gpusprint:local"

echo "Building image ${IMAGE_NAME} directly inside minikube..."
# Build natively inside the minikube container's docker socket to bypass cross-platform loading issues
minikube image build -t "${IMAGE_NAME}" -f Dockerfile .

echo "Restarting gpusprint daemonset to pick up the new image..."
if command -v kubectl &> /dev/null; then
    kubectl -n gpusprint-system rollout restart daemonset gpusprint
else
    minikube kubectl -- -n gpusprint-system rollout restart daemonset gpusprint
fi

echo "Done! The new image is built and the daemonset is restarting."
