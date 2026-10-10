#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/custom_server.sh"

AMAQUET_TEST_TMP="$(mktemp -d -t amaquet-upload-expiry.XXXXXX)"
port="$(free_port)"
admin_port="$(free_port)"
cfg="$AMAQUET_TEST_TMP/amaquet.json"

cat >"$cfg" <<JSON
{"protocol":{"host":"127.0.0.1","port":$port,"tls":false,"require_auth":false},"admin":{"host":"127.0.0.1","port":$admin_port,"metrics_require_auth":true,"max_sessions":100,"auth_failures_per_minute":20},"security":{"bootstrap_token":"chaos"},"persistence":{"aof_enabled":false,"aof_path":"$AMAQUET_TEST_TMP/db.aof","fsync":"no"},"limits":{"max_connections":64,"max_payload_bytes":8388608,"max_memory_bytes":67108864,"eviction_policy":"noeviction","max_inflight_requests":64,"max_inflight_per_connection":8,"max_inflight_payload_bytes":16777216,"max_pending_upload_bytes":8388608,"upload_ttl_ms":1000},"compression":{"enabled":false,"algorithm":"none","min_bytes":0,"min_savings_percent":0},"data_dir":"$AMAQUET_TEST_TMP/data"}
JSON

custom_start_server "$cfg" "$port" "$admin_port"

begin="$(amaquet BLOB_BEGIN '{"key":"abandoned","total_size":5}')"
id="$(printf '%s' "$begin" | python3 -c 'import json,sys;print(json.load(sys.stdin)["upload_id"])')"
sleep 2.2

assert_command_fails amaquet BLOB_CHUNK "{\"upload_id\":\"$id\",\"offset\":0,\"data\":\"aGVsbG8=\"}"
custom_stop_server
rm -rf "$AMAQUET_TEST_TMP"

log "ABANDONED UPLOAD PASS"
