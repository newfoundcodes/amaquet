#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/custom_server.sh"
: "${AMAQUET_RESTART_ITERATIONS:=5}"

AMAQUET_TEST_TMP="$(mktemp -d -t amaquet-kill-restart.XXXXXX)"
port="$(free_port)"
admin_port="$(free_port)"
cfg="$AMAQUET_TEST_TMP/amaquet.json"

cat >"$cfg" <<JSON
{"protocol":{"host":"127.0.0.1","port":$port,"tls":false,"require_auth":false,"read_timeout_ms":5000,"write_timeout_ms":5000},"admin":{"host":"127.0.0.1","port":$admin_port,"metrics_require_auth":true,"max_sessions":100,"auth_failures_per_minute":20},"security":{"bootstrap_token":"chaos"},"persistence":{"aof_enabled":true,"aof_path":"$AMAQUET_TEST_TMP/db.aof","fsync":"always"},"limits":{"max_connections":128,"max_payload_bytes":8388608,"max_memory_bytes":0,"eviction_policy":"noeviction","max_inflight_requests":128,"max_inflight_per_connection":16,"max_inflight_payload_bytes":33554432,"max_pending_upload_bytes":67108864,"upload_ttl_ms":5000},"compression":{"enabled":true,"algorithm":"auto","min_bytes":1024,"min_savings_percent":8},"data_dir":"$AMAQUET_TEST_TMP/data"}
JSON

for ((i = 1; i <= AMAQUET_RESTART_ITERATIONS; i++)); do
    custom_start_server "$cfg" "$port" "$admin_port"
    amaquet SET "{\"key\":\"chaos:durable\",\"type\":\"integer\",\"value\":$i}" >/dev/null

    kill -KILL "$AMAQUET_SERVER_PID" 2>/dev/null || true
    wait "$AMAQUET_SERVER_PID" 2>/dev/null || true

    AMAQUET_OWN_SERVER=0
    AMAQUET_SERVER_PID=""
    custom_start_server "$cfg" "$port" "$admin_port"

    out="$(amaquet GET '{"key":"chaos:durable"}')"
    assert_contains "$out" "\"value\": $i"
    custom_stop_server
done

rm -rf "$AMAQUET_TEST_TMP"
log "KILL/RESTART AOF PASS iterations=$AMAQUET_RESTART_ITERATIONS"
