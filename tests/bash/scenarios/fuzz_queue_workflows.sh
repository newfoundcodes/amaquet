#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/scenario.sh"

scenario_init

fq="$(scenario_key fifo)"
ls="$(scenario_key lifo)"
pq="$(scenario_key prio)"
rq="$(scenario_key reliable)"
dq="$(scenario_key delayed)"

create "$fq" fifo_queue '{}'
create "$ls" lifo_stack '{}'
create "$pq" priority_queue '{}'
create "$rq" reliable_queue '{"max_attempts":3}'
create "$dq" delayed_queue '{}'

for i in 1 2 3 4 5; do
    op "$fq" ENQUEUE "{\"value\":{\"type\":\"integer\",\"value\":$i}}" >/dev/null
    op "$ls" PUSH "{\"value\":{\"type\":\"integer\",\"value\":$i}}" >/dev/null
done

assert_contains "$(op "$fq" DEQUEUE '{}')" '"value": 1'
assert_contains "$(op "$ls" POP '{}')" '"value": 5'

op "$pq" ENQUEUE '{"value":{"type":"utf8_string","value":"low"},"priority":10}' >/dev/null
op "$pq" ENQUEUE '{"value":{"type":"utf8_string","value":"high"},"priority":1}' >/dev/null

assert_contains "$(op "$pq" DEQUEUE '{}')" low

op "$rq" ENQUEUE '{"id":"job","value":{"type":"utf8_string","value":"work"}}' >/dev/null
claim="$(op "$rq" CLAIM '{"lease_ms":50}')"

assert_contains "$claim" job
assert_true "$(op "$rq" NACK '{"id":"job"}' | json_extract x)"
assert_contains "$(op "$rq" CLAIM '{"lease_ms":50}')" job
assert_true "$(op "$rq" ACK '{"id":"job"}' | json_extract x)"

future=$(($(python3 -c 'import time; print(time.time_ns() // 1_000_000)') + 120))
op "$dq" ENQUEUE "{\"id\":\"later\",\"value\":{\"type\":\"utf8_string\",\"value\":\"x\"},\"available_at_ms\":$future}" >/dev/null

early="$(op "$dq" POP_READY '{}' | json_extract x)"
[[ -z "$early" || "$early" == None || "$early" == null ]] || die "delayed item became ready too early: $early"

sleep 0.15
assert_contains "$(op "$dq" POP_READY '{}')" later

server_healthy
echo "PASS queue workflows"
