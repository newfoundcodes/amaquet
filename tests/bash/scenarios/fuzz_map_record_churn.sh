#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/scenario.sh"

scenario_init
h="$(scenario_key hash)"
o="$(scenario_key ordered)"
m="$(scenario_key multi)"

create "$h" hashmap '{}'
create "$o" ordered_map '{}'
create "$m" multimap '{}'

for ((i = 0; i < AMAQUET_SCENARIO_ITERATIONS; i++)); do
    op "$h" SET "{\"field\":\"f$i\",\"value\":{\"type\":\"integer\",\"value\":$i}}" >/dev/null
    op "$o" SET "{\"field\":\"f$i\",\"value\":{\"type\":\"utf8_string\",\"value\":\"v$i\"}}" >/dev/null
    op "$m" ADD "{\"field\":\"bucket$((i % 3))\",\"value\":{\"type\":\"integer\",\"value\":$i}}" >/dev/null
done

assert_eq "$(op "$h" KEYS '{}' | json_len)" "$AMAQUET_SCENARIO_ITERATIONS"
assert_eq "$(op "$o" ENTRIES '{}' | json_len)" "$AMAQUET_SCENARIO_ITERATIONS"
assert_nonempty "$(op "$m" GET '{"field":"bucket0"}' | json_len)"

for ((i = 0; i < AMAQUET_SCENARIO_ITERATIONS; i += 2)); do op "$h" DELETE "{\"field\":\"f$i\"}" >/dev/null; done
server_healthy

echo "PASS map/record churn"
