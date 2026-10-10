#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/scenario.sh"

scenario_init
k="$(scenario_key conditional)"

for ((i = 0; i < AMAQUET_SCENARIO_ITERATIONS; i++)); do
    amaquet DEL "{\"keys\":[\"$k\"]}" >/dev/null || true
    amaquet SET "{\"key\":\"$k\",\"type\":\"integer\",\"value\":$i,\"nx\":true}" >/dev/null
    assert_command_fails amaquet SET "{\"key\":\"$k\",\"type\":\"integer\",\"value\":999,\"nx\":true}"
    amaquet SET "{\"key\":\"$k\",\"type\":\"integer\",\"value\":$((i + 1)),\"xx\":true}" >/dev/null
    assert_eq "$(amaquet GET "{\"key\":\"$k\"}" | python3 -c 'import json,sys;print(json.load(sys.stdin)["value"])')" "$((i + 1))"
done

amaquet DEL "{\"keys\":[\"$k\"]}" >/dev/null
server_healthy

echo "PASS conditional writes iterations=$AMAQUET_SCENARIO_ITERATIONS seed=$AMAQUET_FUZZ_SEED"
