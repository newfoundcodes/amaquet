#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/custom_server.sh"
: "${AMAQUET_MEMORY_WRITES:=200}"

AMAQUET_TEST_TMP="$(mktemp -d -t amaquet-maxmemory.XXXXXX)"
port="$(free_port)"
admin_port="$(free_port)"
token='memory-chaos'
cfg="$AMAQUET_TEST_TMP/amaquet.json"

cat >"$cfg" <<JSON
{"protocol":{"host":"127.0.0.1","port":$port,"tls":false,"require_auth":false},"admin":{"host":"127.0.0.1","port":$admin_port,"metrics_require_auth":true,"max_sessions":100,"auth_failures_per_minute":20},"security":{"bootstrap_token":"$token"},"persistence":{"aof_enabled":false,"aof_path":"$AMAQUET_TEST_TMP/db.aof","fsync":"no"},"limits":{"max_connections":128,"max_payload_bytes":8388608,"max_memory_bytes":262144,"eviction_policy":"allkeys-lru","max_inflight_requests":128,"max_inflight_per_connection":16,"max_inflight_payload_bytes":33554432,"max_pending_upload_bytes":67108864,"upload_ttl_ms":5000},"compression":{"enabled":false,"algorithm":"none","min_bytes":0,"min_savings_percent":0},"data_dir":"$AMAQUET_TEST_TMP/data"}
JSON

custom_start_server "$cfg" "$port" "$admin_port"
payload="$(
    python3 - <<'PY'
print('x'*4096)
PY
)"

for ((i = 0; i < AMAQUET_MEMORY_WRITES; i++)); do
    amaquet SET "{\"key\":\"mem:$i\",\"type\":\"utf8_string\",\"value\":\"$payload\"}" >/dev/null
done

metrics="$(admin_curl "$token" GET /metrics)"
evicted="$(printf '%s\n' "$metrics" | awk '$1=="amaquet_evicted_keys_total" {print $2}')"

[[ -n "$evicted" && "$evicted" -gt 0 ]] || die "expected memory eviction, metrics=$metrics"
amaquet PING '{}' >/dev/null

custom_stop_server
rm -rf "$AMAQUET_TEST_TMP"
log "MAXMEMORY PASS writes=$AMAQUET_MEMORY_WRITES evicted=$evicted"
