#!/usr/bin/env bash
# The whole story on a real cluster: a Spring Boot pod gets OOMKilled with a
# half-empty heap, jvm-fit explains why, and its fix makes the pod stable.
# Every result below is read from the cluster. Run demo/setup.sh first.
set -euo pipefail
source "$(dirname "$0")/env.sh"
cd "$DEMO_DIR"
PATH="$REPO_DIR/bin:$PATH"

say() { printf '\n\033[1;36m# %s\033[0m\n' "$*"; sleep "${DEMO_PAUSE:-1}"; }
show() { printf '\033[1m$ %s\033[0m\n' "$*"; eval "$*"; }
# Time to read the output in the recording.
hold() { sleep "${DEMO_READ:-0}"; }
die() { printf '\033[1;31mDemo failed: %s\033[0m\n' "$*" >&2; exit 1; }

pod() { kubectl get pod -l app=orders -o name --field-selector=status.phase=Running | head -1; }
field() { kubectl get "$(pod)" -o jsonpath="{.status.containerStatuses[0].$1}" 2>/dev/null; }

# Wait until the old pods are gone, so every read below is about the new one.
rollout() {
  kubectl rollout status deployment/orders --timeout=180s >/dev/null
  while [ "$(kubectl get pod -l app=orders -o name | wc -l)" -gt 1 ]; do sleep 1; done
}

say "A Spring Boot service: the heap is set to 450Mi, the memory limit is 512Mi"
show "grep -E 'Xmx|memory:' k8s/orders-bad.yaml"
show "kubectl apply -f k8s/orders-bad.yaml"
rollout

say "It keeps 260Mi in the heap and 140Mi in direct buffers. Waiting for a crash..."
for _ in $(seq 180); do
  [ "$(field lastState.terminated.reason)" = OOMKilled ] && break
  sleep 1
done
[ "$(field lastState.terminated.reason)" = OOMKilled ] || die "no OOMKill in 3 minutes"
show "kubectl get pods"
printf 'last state: %s, exit code %s\n' "$(field lastState.terminated.reason)" "$(field lastState.terminated.exitCode)"

say "OOMKilled, but look at the heap in the last line it logged"
show "kubectl logs $(pod) --previous | grep '^heap' | tail -1"
hold

say "The heap was not full. Ask jvm-fit, which needs only the manifest"
show "jvm-fit scan k8s/orders-bad.yaml || true"
hold

say "Apply its fix: the same app with the same load"
show "diff k8s/orders-bad.yaml k8s/orders-fixed.yaml | grep -E '^[<>] +(value|memory)' || true"
show "jvm-fit scan k8s/orders-fixed.yaml | tail -1"
show "kubectl apply -f k8s/orders-fixed.yaml"
rollout

say "Waiting for the full load, then one more minute..."
for _ in $(seq 180); do
  kubectl logs "$(pod)" 2>/dev/null | grep -q '^load complete' && break
  sleep 1
done
kubectl logs "$(pod)" | grep -q '^load complete' || die "the load did not complete"
sleep "${DEMO_HOLD:-60}"

restarts="$(field restartCount)"
[ "$restarts" = 0 ] || die "$restarts restarts"
show "kubectl get pods"
show "kubectl logs $(pod) | grep '^load complete'"
show "kubectl logs $(pod) | grep '^heap' | tail -1"
peak="$(kubectl exec "$(pod)" -- sh -c 'cat /sys/fs/cgroup/memory.peak 2>/dev/null || cat /sys/fs/cgroup/memory/memory.max_usage_in_bytes')"
printf 'peak memory: %dMi of the 832Mi limit\n' "$((peak / 1048576))"
hold

say "Stable: 0 restarts with the full load"
