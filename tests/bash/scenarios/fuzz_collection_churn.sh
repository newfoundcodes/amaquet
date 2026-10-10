#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/scenario.sh"

scenario_init
l="$(scenario_key list)"
s="$(scenario_key set)"
d="$(scenario_key deque)"
r="$(scenario_key ring)"

create "$l" list '{}'
create "$s" set '{}'
create "$d" deque '{"capacity":10000}'
create "$r" ring_buffer '{"capacity":10}'

for ((i = 0; i < AMAQUET_SCENARIO_ITERATIONS; i++)); do
    v="v-$i"
    op "$l" PUSH_BACK "{\"value\":{\"type\":\"utf8_string\",\"value\":\"$v\"}}" >/dev/null
    op "$s" ADD "{\"value\":{\"type\":\"integer\",\"value\":$((i % 7))}}" >/dev/null
    op "$d" PUSH_FRONT "{\"value\":{\"type\":\"integer\",\"value\":$i}}" >/dev/null
    op "$r" PUSH "{\"value\":{\"type\":\"integer\",\"value\":$i}}" >/dev/null
done

assert_eq "$(op "$l" LEN '{}' | json_extract x)" "$AMAQUET_SCENARIO_ITERATIONS"
setlen="$(op "$s" MEMBERS '{}' | json_len)"
expected=$((AMAQUET_SCENARIO_ITERATIONS < 7 ? AMAQUET_SCENARIO_ITERATIONS : 7))

assert_eq "$setlen" "$expected"
assert_eq "$(op "$d" LEN '{}' | json_extract x)" "$AMAQUET_SCENARIO_ITERATIONS"

ringlen="$(op "$r" LEN '{}' | json_extract x)"
expected_ring=$((AMAQUET_SCENARIO_ITERATIONS < 10 ? AMAQUET_SCENARIO_ITERATIONS : 10))

assert_eq "$ringlen" "$expected_ring"
server_healthy

echo "PASS collection churn"
