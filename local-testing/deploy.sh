#!/usr/bin/env bash
# Deploy gpusprint + fake device plugin + test workloads.
# Supports both local builds and production (GHCR) images.
set -euo pipefail
cd "$(dirname "$0")/.."

KUBECTL="kubectl"
command -v kubectl &>/dev/null || KUBECTL="minikube kubectl --"
command -v helm &>/dev/null || { echo "helm not found"; exit 1; }

# Pass USE_LOCAL=1 to build & load local images; default is production (GHCR).
# Pass GPUSPRINT_TAG to override the image tag (default: latest).
USE_LOCAL="${USE_LOCAL:-0}"
GPUSPRINT_TAG="${GPUSPRINT_TAG:-latest}"

NODES=$($KUBECTL get nodes --no-headers | wc -l | tr -d ' ')

if [[ "$USE_LOCAL" == "1" ]]; then
  echo "Building local images..."
  podman build -t fake-gpu-device-plugin:local -f local-testing/fake-device-plugin/Dockerfile .
  podman build -t gpusprint:local -f Dockerfile .
  podman tag localhost/fake-gpu-device-plugin:local docker.io/library/fake-gpu-device-plugin:local
  podman tag localhost/gpusprint:local docker.io/library/gpusprint:local
  echo "Loading images into minikube..."
  minikube image load docker.io/library/fake-gpu-device-plugin:local
  minikube image load docker.io/library/gpusprint:local
  HELM_VALUES_FILE="deploy/helm/values-local.yaml"
else
  echo "Using production images from GHCR..."
  # Only the fake device plugin needs a local build
  podman build -t fake-gpu-device-plugin:local -f local-testing/fake-device-plugin/Dockerfile .
  podman tag localhost/fake-gpu-device-plugin:local docker.io/library/fake-gpu-device-plugin:local
  minikube image load docker.io/library/fake-gpu-device-plugin:local
  HELM_VALUES_FILE="deploy/helm/values.yaml"
fi

echo "Deploying to ${NODES} nodes..."
$KUBECTL apply -f local-testing/fake-device-plugin/daemonset.yaml
$KUBECTL -n kube-system rollout status daemonset/fake-gpu-device-plugin --timeout=300s
sleep 5

$KUBECTL create namespace gpusprint-system --dry-run=client -o yaml | $KUBECTL apply -f -
helm upgrade --install gpusprint ./deploy/helm \
    --namespace gpusprint-system \
    -f "$HELM_VALUES_FILE" \
    --set image.tag="$GPUSPRINT_TAG" \
    --set config.developmentMode="true" \
    --set config.numGPUs="12" \
    --wait --timeout 300s

$KUBECTL apply -f local-testing/manifests/test-pods.yaml
$KUBECTL apply -f local-testing/manifests/otel-collector.yaml

echo "Done. ${NODES} nodes ready."
echo "  kubectl get pods -A -o wide"
echo "  kubectl -n gpusprint-system port-forward daemonset/gpusprint 9400:9400"
echo "  curl http://localhost:9400/metrics"
