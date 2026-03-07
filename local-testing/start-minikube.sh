#!/usr/bin/env bash
# Start multi-node minikube cluster for local GPU testing.
# Usage: ./local-testing/start-minikube.sh [NODES]   (default: 10)
set -euo pipefail

NODES="${1:-${NODES:-10}}"

# Ensure podman machine has enough RAM (~1800 MB/node — minikube minimum)
if command -v podman &> /dev/null; then
    PODMAN_MEM=$(podman machine inspect --format '{{.Resources.Memory}}' 2>/dev/null || echo 0)
    NEEDED_MB=$(( NODES * 1800 + 2048 ))
    if (( PODMAN_MEM < NEEDED_MB )); then
        echo "Podman machine has ${PODMAN_MEM} MB, need ${NEEDED_MB} MB. Resizing..."
        podman machine stop
        podman machine set --memory "${NEEDED_MB}" --cpus 12
        podman machine start
    fi
fi

echo "Starting minikube with ${NODES} nodes..."
minikube start --driver=podman --nodes="${NODES}" --memory=1800m

KUBECTL="kubectl"
command -v kubectl &>/dev/null || KUBECTL="minikube kubectl --"
$KUBECTL wait --for=condition=Ready nodes --all --timeout=600s
$KUBECTL get nodes
