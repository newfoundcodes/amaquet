#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/scenario.sh"

scenario_init

lease="$(scenario_key lease)"
lock="$(scenario_key lock)"
sem="$(scenario_key sem)"
rate="$(scenario_key rate)"

create "$lease" lease '{}'
create "$lock" lock '{}'
create "$sem" semaphore '{"limit":2}'
create "$rate" rate_limit_bucket '{"capacity":3,"refill_per_second":1}'

assert_true "$(op "$lease" ACQUIRE '{"owner":"a","token":"t1","ttl_ms":100}' | json_extract x)"
assert_false "$(op "$lease" ACQUIRE '{"owner":"b","token":"t2","ttl_ms":100}' | json_extract x)"
assert_true "$(op "$lease" RENEW '{"token":"t1","ttl_ms":100}' | json_extract x)"
assert_true "$(op "$lease" RELEASE '{"token":"t1"}' | json_extract x)"
assert_true "$(op "$lock" TRY_LOCK '{"owner":"a","token":"lt","ttl_ms":100}' | json_extract x)"
assert_false "$(op "$lock" TRY_LOCK '{"owner":"b","token":"lt2","ttl_ms":100}' | json_extract x)"
assert_true "$(op "$lock" RENEW '{"token":"lt","ttl_ms":100}' | json_extract x)"
assert_true "$(op "$lock" UNLOCK '{"token":"lt"}' | json_extract x)"
assert_true "$(op "$sem" TRY_ACQUIRE '{}' | json_extract x)"
assert_true "$(op "$sem" TRY_ACQUIRE '{}' | json_extract x)"
assert_false "$(op "$sem" TRY_ACQUIRE '{}' | json_extract x)"
assert_true "$(op "$sem" RELEASE '{}' | json_extract x)"

for i in 1 2 3; do assert_true "$(op "$rate" ALLOW '{"cost":1}' | json_extract x)"; done
assert_false "$(op "$rate" ALLOW '{"cost":1}' | json_extract x)"

server_healthy
echo "PASS synchronization workflows"
