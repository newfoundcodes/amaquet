#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/scenario.sh"

scenario_init

q="$(scenario_key deadletter)"
create "$q" reliable_queue '{"max_attempts":3}'
op "$q" ENQUEUE '{"id":"bad-job","value":{"type":"utf8_string","value":"payload"}}' >/dev/null

for i in 1 2 3; do
    assert_contains "$(op "$q" CLAIM '{"lease_ms":1000}')" bad-job
    assert_true "$(op "$q" NACK '{"id":"bad-job"}' | json_extract x)"
done

stats="$(op "$q" STATS '{}')"

assert_eq "$(printf '%s' "$stats" | python3 -c 'import json,sys;print(json.load(sys.stdin)["dead"])')" 1
assert_eq "$(printf '%s' "$stats" | python3 -c 'import json,sys;print(json.load(sys.stdin)["ready"])')" 0

server_healthy
echo "PASS reliable queue dead-letter path"
