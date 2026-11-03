# Shared settings. The demo has its own kubeconfig and never changes yours.
DEMO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(dirname "$DEMO_DIR")"
CLUSTER=jvm-fit-demo
IMAGE=jvm-fit-demo/orders:1.0.0
# 1.33 still runs on hosts with cgroup v1. Set KIND_IMAGE to use another one.
KIND_IMAGE="${KIND_IMAGE:-kindest/node:v1.33.1}"
export KUBECONFIG="$DEMO_DIR/.kubeconfig"
