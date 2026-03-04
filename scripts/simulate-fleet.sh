#!/usr/bin/env bash
set -euo pipefail

MODE="${1:-start}"

if [[ "$MODE" == "start" ]]; then
    echo "Starting 100 simulated nodes..."
    if command -v kubectl &> /dev/null; then
        kubectl apply -f deploy/scale-test.yaml
    else
        minikube kubectl -- apply -f deploy/scale-test.yaml
    fi
    echo "Deployment applied! You can check the pods with: kubectl -n gpusprint-system get pods"
elif [[ "$MODE" == "stop" ]]; then
    echo "Stopping simulated nodes..."
    if command -v kubectl &> /dev/null; then
        kubectl delete -f deploy/scale-test.yaml --ignore-not-found
    else
        minikube kubectl -- delete -f deploy/scale-test.yaml --ignore-not-found
    fi
    echo "Simulation stopped."
else
    echo "Usage: ./simulate-fleet.sh [start|stop]"
    exit 1
fi
