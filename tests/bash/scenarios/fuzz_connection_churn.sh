#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/scenario.sh"

scenario_init
workers="${AMAQUET_CONCURRENCY:-12}"
per="$AMAQUET_SCENARIO_ITERATIONS"

pids=()
for ((w = 0; w < workers; w++)); do
    (for ((i = 0; i < per; i++)); do "$AMAQUET_CLI" -uri "$AMAQUET_URI" -timeout 3s PING '{}' >/dev/null; done) &
    pids+=("$!")
done

for p in "${pids[@]}"; do wait "$p"; done
server_healthy

echo "PASS connection churn connections=$((workers * per))"
