#!/usr/bin/env bash
set -euo pipefail

if ! command -v minikube &> /dev/null; then
    echo "minikube not found. Installing for ARM64 macOS..."
    curl -LO https://github.com/kubernetes/minikube/releases/latest/download/minikube-darwin-arm64
    sudo install minikube-darwin-arm64 /usr/local/bin/minikube
    rm minikube-darwin-arm64
fi

echo "Starting minikube with podman driver..."
# IMPORTANT: Podman machine MUST be running in rootful mode, otherwise nested containers (etcd, kube-apiserver, etc.)
# fail to start with "error setting rlimit type 7: operation not permitted".
# If minikube hangs on "Booting up control plane...", run: podman machine stop && podman machine set --rootful && podman machine start
minikube start --driver=podman --docker-opt="default-ulimit=nproc=65535:65535"

echo "Waiting for minikube nodes to be ready..."
# minikube can download the appropriate version of kubectl if it's not installed globally
if command -v kubectl &> /dev/null; then
    kubectl wait --for=condition=Ready nodes --all --timeout=120s
else
    minikube kubectl -- wait --for=condition=Ready nodes --all --timeout=120s
fi

echo "minikube started successfully!"
