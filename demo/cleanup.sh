#!/usr/bin/env bash
# Delete the demo cluster.
set -euo pipefail
source "$(dirname "$0")/env.sh"
kind delete cluster --name "$CLUSTER"
rm -f "$KUBECONFIG"
