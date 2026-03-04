#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

MODE="${1:-start}"

KUBECTL="kubectl"
if ! command -v kubectl &>/dev/null; then
    KUBECTL="minikube kubectl --"
fi

if [[ "$MODE" == "start" ]]; then
    echo "Starting 100 simulated nodes..."
    $KUBECTL apply -f local-testing/manifests/scale-test.yaml
    echo "Deployment applied! You can check the pods with: kubectl -n gpusprint-system get pods"
elif [[ "$MODE" == "stop" ]]; then
    echo "Stopping simulated nodes..."
    $KUBECTL delete -f local-testing/manifests/scale-test.yaml --ignore-not-found
    echo "Simulation stopped."
else
    echo "Usage: ./simulate-fleet.sh [start|stop]"
    exit 1
fi
