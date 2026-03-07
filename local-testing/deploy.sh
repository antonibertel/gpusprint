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
  podman build -t gpusprint:local -f Dockerfile .
  podman build -t fake-gpu-device-plugin:local -f local-testing/fake-device-plugin/Dockerfile .
  echo "Loading images into minikube (via tar for multi-node)..."
  podman save gpusprint:local -o /tmp/gpusprint.tar
  podman save fake-gpu-device-plugin:local -o /tmp/fake-gpu-device-plugin.tar
  minikube image load /tmp/gpusprint.tar
  minikube image load /tmp/fake-gpu-device-plugin.tar
  rm -f /tmp/gpusprint.tar /tmp/fake-gpu-device-plugin.tar
  HELM_VALUES_FILE="deploy/helm/values-local.yaml"
  GPUSPRINT_TAG="local"
else
  echo "Using production images from GHCR..."
  # Only the fake device plugin needs a local build
  podman build -t fake-gpu-device-plugin:local -f local-testing/fake-device-plugin/Dockerfile .
  podman save fake-gpu-device-plugin:local -o /tmp/fake-gpu-device-plugin.tar
  minikube image load /tmp/fake-gpu-device-plugin.tar
  rm -f /tmp/fake-gpu-device-plugin.tar
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
