#!/usr/bin/env bash
# Create the kind cluster, build the app image and build jvm-fit.
set -euo pipefail
source "$(dirname "$0")/env.sh"

for tool in docker kind kubectl go; do
  command -v "$tool" >/dev/null || { echo "$tool is not installed" >&2; exit 1; }
done

if ! kind get clusters 2>/dev/null | grep -qx "$CLUSTER"; then
  kind create cluster --name "$CLUSTER" --image "$KIND_IMAGE" --wait 120s
fi
kind export kubeconfig --name "$CLUSTER" >/dev/null 2>&1
docker build -t "$IMAGE" "$DEMO_DIR/app"
kind load docker-image "$IMAGE" --name "$CLUSTER"
(cd "$REPO_DIR" && go build -o bin/jvm-fit ./cmd/jvm-fit)
kubectl delete deployment orders --ignore-not-found >/dev/null
echo "Ready. Run demo/run.sh"
