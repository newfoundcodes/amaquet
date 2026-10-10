#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/custom_server.sh"
need curl

AMAQUET_TEST_TMP="$(mktemp -d -t amaquet-key-expiry.XXXXXX)"
port="$(free_port)"
admin_port="$(free_port)"
token='bootstrap-expiry'

cat >"$AMAQUET_TEST_TMP/amaquet.json" <<JSON
{"protocol":{"host":"127.0.0.1","port":$port,"tls":false,"require_auth":true},"admin":{"host":"127.0.0.1","port":$admin_port},"security":{"bootstrap_token":"$token"},"persistence":{"aof_enabled":false,"aof_path":"$AMAQUET_TEST_TMP/db.aof","fsync":"no"},"limits":{"max_connections":64,"max_payload_bytes":8388608},"compression":{"enabled":true,"algorithm":"auto","min_bytes":1024,"min_savings_percent":8},"data_dir":"$AMAQUET_TEST_TMP/data"}
JSON

custom_start_server "$AMAQUET_TEST_TMP/amaquet.json" "$port" "$admin_port"

past='2000-01-01T00:00:00Z'
future='2099-01-01T00:00:00Z'
past_json="$(admin_curl "$token" POST /api/api-keys "{\"name\":\"expired\",\"role\":\"developer\",\"expires_at\":\"$past\"}")"
past_secret="$(printf '%s' "$past_json" | python3 -c 'import json,sys;print(json.load(sys.stdin)["secret"])')"
future_json="$(admin_curl "$token" POST /api/api-keys "{\"name\":\"future\",\"role\":\"developer\",\"expires_at\":\"$future\"}")"
future_secret="$(printf '%s' "$future_json" | python3 -c 'import json,sys;print(json.load(sys.stdin)["secret"])')"

if "$AMAQUET_CLI" -uri "amaquet://127.0.0.1:$port" -api-key "$past_secret" PING '{}' >/dev/null 2>&1; then die "expired API key authenticated"; fi

"$AMAQUET_CLI" -uri "amaquet://127.0.0.1:$port" -api-key "$future_secret" PING '{}' >/dev/null
custom_stop_server

rm -rf "$AMAQUET_TEST_TMP"
echo "PASS API key expiry"
