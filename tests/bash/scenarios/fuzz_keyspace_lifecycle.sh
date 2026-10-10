#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/scenario.sh"

scenario_init
p="$(scenario_key keyspace)"
k="$p:item"

setv "$k" integer 1

assert_true "$(amaquet EXISTS "{\"key\":\"$k\"}" | json_extract x)"
assert_eq "$(amaquet TYPE "{\"key\":\"$k\"}" | json_extract x)" integer
assert_contains "$(amaquet GET "{\"key\":\"$k\"}")" '"value": 1'
assert_command_fails amaquet SET "{\"key\":\"$k\",\"type\":\"integer\",\"value\":2,\"nx\":true}"
amaquet SET "{\"key\":\"$k\",\"type\":\"integer\",\"value\":3,\"xx\":true}" >/dev/null

assert_command_fails amaquet SET "{\"key\":\"$p:missing\",\"type\":\"integer\",\"value\":1,\"xx\":true}"
assert_eq "$(amaquet GET "{\"key\":\"$k\"}" | python3 -c 'import json,sys;print(json.load(sys.stdin)["value"])')" 3
amaquet EXPIRE "{\"key\":\"$k\",\"ttl_ms\":1000}" >/dev/null

ttl="$(amaquet TTL "{\"key\":\"$k\"}" | json_extract x)"
((ttl > 0 && ttl <= 1000)) || die "TTL out of range: $ttl"
amaquet PERSIST "{\"key\":\"$k\"}" >/dev/null

assert_eq "$(amaquet TTL "{\"key\":\"$k\"}" | json_extract x)" -1
keys_args="{\"prefix\":\"$p:\",\"limit\":10}"
keys_json="$(amaquet KEYS "$keys_args")"

assert_contains "$keys_json" "$k"
assert_contains "$(amaquet MEMORY "{\"key\":\"$k\"}")" '"version"'

assert_eq "$(amaquet DEL "{\"keys\":[\"$k\"]}" | json_extract x)" 1
assert_false "$(amaquet EXISTS "{\"key\":\"$k\"}" | json_extract x)"

server_healthy
echo "PASS keyspace lifecycle"
