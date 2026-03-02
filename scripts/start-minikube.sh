#!/usr/bin/env bash
set -euo pipefail

if ! command -v minikube &> /dev/null; then
    echo "minikube not found. Installing for ARM64 macOS..."
    curl -LO https://github.com/kubernetes/minikube/releases/latest/download/minikube-darwin-arm64
    sudo install minikube-darwin-arm64 /usr/local/bin/minikube
    rm minikube-darwin-arm64
fi

echo "Starting minikube with podman driver..."
minikube start --driver=podman

echo "Waiting for minikube nodes to be ready..."
# minikube can download the appropriate version of kubectl if it's not installed globally
if command -v kubectl &> /dev/null; then
    kubectl wait --for=condition=Ready nodes --all --timeout=120s
else
    minikube kubectl -- wait --for=condition=Ready nodes --all --timeout=120s
fi

echo "minikube started successfully!"
