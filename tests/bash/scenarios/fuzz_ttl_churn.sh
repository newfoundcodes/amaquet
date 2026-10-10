#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/scenario.sh"

scenario_init
p="scenario:ttl:$$"
count=$((AMAQUET_SCENARIO_ITERATIONS * 3))

for ((i = 0; i < count; i++)); do
    ttl=$((50 + (i % 5) * 20))
    amaquet SET "{\"key\":\"$p:$i\",\"type\":\"integer\",\"value\":$i,\"ttl_ms\":$ttl}" >/dev/null
done

sleep 0.20

left="$(amaquet KEYS "{\"prefix\":\"$p:\",\"limit\":10000}" | json_len)"
assert_eq "$left" 0

server_healthy
echo "PASS TTL churn keys=$count"
