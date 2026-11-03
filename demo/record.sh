#!/usr/bin/env bash
# Record demo/run.sh to docs/demo.gif. Needs asciinema and agg. Waits longer
# than 4 seconds are cut to 4 seconds in the GIF; nothing else changes.
set -euo pipefail
source "$(dirname "$0")/env.sh"
cast="$(mktemp -t jvm-fit-demo).cast"
kubectl delete deployment orders --ignore-not-found --wait >/dev/null
DEMO_PAUSE=2 DEMO_READ=4 asciinema rec --overwrite --quiet --window-size 124x38 --command "$DEMO_DIR/run.sh" "$cast"
agg --idle-time-limit 4 --last-frame-duration 6 --font-size 14 "$cast" "$REPO_DIR/docs/demo.gif"
rm -f "$cast"
ls -lh "$REPO_DIR/docs/demo.gif"
