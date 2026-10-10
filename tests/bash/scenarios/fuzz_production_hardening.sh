#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/scenario.sh"

scenario_init
p="$(scenario_key hardening)"

# Cursor SCAN: collect a prefixed keyspace in small pages.
for i in $(seq 1 23); do setv "$p:scan:$i" integer "$i"; done

cursor=""
seen=0

for _ in $(seq 1 20); do
    out="$(amaquet SCAN "{\"prefix\":\"$p:scan:\",\"cursor\":\"$cursor\",\"count\":4}")"
    n="$(printf '%s' "$out" | python3 -c 'import json,sys; print(len(json.load(sys.stdin)["keys"]))')"

    seen=$((seen + n))
    cursor="$(printf '%s' "$out" | python3 -c 'import json,sys; print(json.load(sys.stdin)["cursor"])')"

    [[ -z "$cursor" ]] && break
done

assert_eq "$seen" 23 "SCAN must return each matching key once"

# Optimistic CAS and stale-version conflict.
setv "$p:cas" integer 1
ver="$(amaquet GET "{\"key\":\"$p:cas\"}" | python3 -c 'import json,sys; print(json.load(sys.stdin)["version"])')"
amaquet CAS "{\"key\":\"$p:cas\",\"expected_version\":$ver,\"type\":\"integer\",\"value\":2,\"ttl_ms\":5000}" >/dev/null

assert_contains "$(amaquet GET "{\"key\":\"$p:cas\"}")" '"value": 2'
assert_command_fails amaquet CAS "{\"key\":\"$p:cas\",\"expected_version\":$ver,\"type\":\"integer\",\"value\":3}"

# Batch preserves request/result order and can mix writes and reads.
out="$(amaquet BATCH "{\"requests\":[{\"command\":\"SET\",\"args\":{\"key\":\"$p:batch\",\"type\":\"utf8_string\",\"value\":\"ok\"}},{\"command\":\"GET\",\"args\":{\"key\":\"$p:batch\"}}]}")"
assert_eq "$(printf '%s' "$out" | python3 -c 'import json,sys; x=json.load(sys.stdin); print(len(x))')" 2
assert_contains "$out" 'ok'

# Chunked blob upload keeps individual requests small and reconstructs the value.
begin="$(amaquet BLOB_BEGIN "{\"key\":\"$p:blob\",\"total_size\":12}")"
upload_id="$(printf '%s' "$begin" | python3 -c 'import json,sys; print(json.load(sys.stdin)["upload_id"])')"

amaquet BLOB_CHUNK "{\"upload_id\":\"$upload_id\",\"offset\":0,\"data\":\"SGVsbG8g\"}" >/dev/null
amaquet BLOB_CHUNK "{\"upload_id\":\"$upload_id\",\"offset\":6,\"data\":\"V29ybGQh\"}" >/dev/null
amaquet BLOB_COMMIT "{\"upload_id\":\"$upload_id\"}" >/dev/null

assert_contains "$(amaquet GET "{\"key\":\"$p:blob\"}")" 'SGVsbG8gV29ybGQh'

# An expired key cannot be revived by PERSIST.
setv "$p:expired" utf8_string '"gone"'
amaquet EXPIRE "{\"key\":\"$p:expired\",\"ttl_ms\":25}" >/dev/null
sleep 0.05

assert_command_fails amaquet PERSIST "{\"key\":\"$p:expired\"}"
assert_false "$(amaquet EXISTS "{\"key\":\"$p:expired\"}" | json_extract x)"

server_healthy
echo "PASS production hardening"
