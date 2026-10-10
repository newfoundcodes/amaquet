#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/scenario.sh"

scenario_init
k="$(scenario_key ts)"

create "$k" timeseries '{"labels":{"env":"fuzz"},"retention_ms":600000}'
base=$(($(date +%s) * 1000))

for ((i = AMAQUET_SCENARIO_ITERATIONS - 1; i >= 0; i--)); do
    ts=$((base + i * 1000))
    op "$k" ADD "{\"timestamp\":$ts,\"value\":$i}" >/dev/null
done

assert_eq "$(op "$k" LEN '{}' | json_extract x)" "$AMAQUET_SCENARIO_ITERATIONS"

range="$(op "$k" RANGE "{\"from\":$base,\"to\":$((base + AMAQUET_SCENARIO_ITERATIONS * 1000))}")"
assert_eq "$(printf '%s' "$range" | json_len)" "$AMAQUET_SCENARIO_ITERATIONS"
assert_nonempty "$(op "$k" LAST '{}' | json_extract x)"

agg="$(op "$k" AGGREGATE "{\"from\":$base,\"to\":$((base + AMAQUET_SCENARIO_ITERATIONS * 1000)),\"bucket_ms\":5000,\"aggregation\":\"avg\"}")"
assert_nonempty "$(printf '%s' "$agg" | json_len)"

server_healthy
echo "PASS timeseries workloads"
