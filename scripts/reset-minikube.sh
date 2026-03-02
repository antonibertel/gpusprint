#!/usr/bin/env bash
set -euo pipefail

echo "Deleting all minikube clusters (reset)..."
minikube delete --all

echo "minikube clusters deleted."
