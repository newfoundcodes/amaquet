#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/scenario.sh"

scenario_init

s="$(scenario_key stream)"
t="$(scenario_key topic)"
e="$(scenario_key event)"

create "$s" stream '{}'
create "$t" persistent_topic '{"max_messages":1000}'
create "$e" event_log '{}'

for ((i = 0; i < AMAQUET_SCENARIO_ITERATIONS; i++)); do
    op "$s" ADD "{\"fields\":{\"n\":{\"type\":\"integer\",\"value\":$i},\"msg\":{\"type\":\"utf8_string\",\"value\":\"m$i\"}}}" >/dev/null
    op "$t" PUBLISH "{\"message_key\":\"m$i\",\"value\":{\"type\":\"integer\",\"value\":$i}}" >/dev/null
    op "$e" APPEND "{\"event_type\":\"evt\",\"aggregate_id\":\"a$((i % 3))\",\"payload\":{\"type\":\"integer\",\"value\":$i}}" >/dev/null
done

assert_eq "$(op "$s" LEN '{}' | json_extract x)" "$AMAQUET_SCENARIO_ITERATIONS"
assert_eq "$(op "$e" LEN '{}' | json_extract x)" "$AMAQUET_SCENARIO_ITERATIONS"
assert_nonempty "$(op "$t" READ '{"from":0,"count":100}' | json_len)"

op "$s" GROUP_CREATE '{"group":"workers"}' >/dev/null
gr="$(op "$s" GROUP_READ '{"group":"workers","consumer":"c1","count":3}')"

assert_nonempty "$(printf '%s' "$gr" | json_len)"
assert_nonempty "$(op "$s" PENDING '{"group":"workers"}' | json_len)"

server_healthy
echo "PASS stream/topic/event workflows"
