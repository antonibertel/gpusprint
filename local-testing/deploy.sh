#!/usr/bin/env bash
# Build and deploy gpusprint + fake device plugin + test workloads.
set -euo pipefail
cd "$(dirname "$0")/.."

KUBECTL="kubectl"
command -v kubectl &>/dev/null || KUBECTL="minikube kubectl --"
command -v helm &>/dev/null || { echo "helm not found"; exit 1; }

NODES=$($KUBECTL get nodes --no-headers | wc -l | tr -d ' ')

echo "Building images..."
podman build -t fake-gpu-device-plugin:local -f local-testing/fake-device-plugin/Dockerfile .
podman build -t gpusprint:local -f Dockerfile .
podman tag localhost/fake-gpu-device-plugin:local docker.io/library/fake-gpu-device-plugin:local
podman tag localhost/gpusprint:local docker.io/library/gpusprint:local
echo "Loading images into minikube..."
minikube image load docker.io/library/fake-gpu-device-plugin:local
minikube image load docker.io/library/gpusprint:local

echo "Deploying to ${NODES} nodes..."
$KUBECTL apply -f local-testing/fake-device-plugin/daemonset.yaml
$KUBECTL -n kube-system rollout status daemonset/fake-gpu-device-plugin --timeout=300s
sleep 5

$KUBECTL create namespace gpusprint-system --dry-run=client -o yaml | $KUBECTL apply -f -
helm upgrade --install gpusprint ./deploy/helm \
    --namespace gpusprint-system \
    -f deploy/helm/values-local.yaml \
    --wait --timeout 300s

$KUBECTL apply -f local-testing/manifests/test-pods.yaml
$KUBECTL apply -f local-testing/manifests/otel-collector.yaml

echo "Done. ${NODES} nodes ready."
echo "  kubectl get pods -A -o wide"
echo "  kubectl -n gpusprint-system port-forward daemonset/gpusprint 9400:9400"
echo "  curl http://localhost:9400/metrics"
