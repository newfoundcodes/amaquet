#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/scenario.sh"

scenario_init
k="$(scenario_key counter)"

create "$k" atomic_counter '{"initial":0}'
workers="${AMAQUET_CONCURRENCY:-8}"
per="$AMAQUET_SCENARIO_ITERATIONS"

pids=()
for ((w = 0; w < workers; w++)); do
    (for ((i = 0; i < per; i++)); do "$AMAQUET_CLI" -uri "$AMAQUET_URI" OP "{\"key\":\"$k\",\"operation\":\"ADD\",\"args\":{\"delta\":1}}" >/dev/null; done) &
    pids+=("$!")
done

for p in "${pids[@]}"; do wait "$p"; done
assert_eq "$(op "$k" LOAD '{}' | json_extract x)" "$((workers * per))"
server_healthy

echo "PASS concurrent atomic counter workers=$workers iterations=$per"
