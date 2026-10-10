#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/custom_server.sh"
need curl

AMAQUET_TEST_TMP="$(mktemp -d -t amaquet-admin-restart.XXXXXX)"
port="$(free_port)"
admin_port="$(free_port)"
token='admin-persist-token'
cfg="$AMAQUET_TEST_TMP/amaquet.json"

cat >"$cfg" <<JSON
{"protocol":{"host":"127.0.0.1","port":$port,"tls":false,"require_auth":true},"admin":{"host":"127.0.0.1","port":$admin_port},"security":{"bootstrap_token":"$token"},"persistence":{"aof_enabled":false,"aof_path":"$AMAQUET_TEST_TMP/db.aof","fsync":"no"},"limits":{"max_connections":64,"max_payload_bytes":8388608},"compression":{"enabled":true,"algorithm":"auto","min_bytes":1024,"min_savings_percent":8},"data_dir":"$AMAQUET_TEST_TMP/data"}
JSON

custom_start_server "$cfg" "$port" "$admin_port"
admin_curl "$token" PUT /api/organization '{"name":"Persist Org","slug":"persist-org"}' >/dev/null
member="$(admin_curl "$token" POST /api/members '{"name":"Dev","email":"dev@example.com","role":"developer"}')"
keyjson="$(admin_curl "$token" POST /api/api-keys '{"name":"persistent-key","role":"developer"}')"
secret="$(printf '%s' "$keyjson" | python3 -c 'import json,sys;print(json.load(sys.stdin)["secret"])')"
custom_stop_server

custom_start_server "$cfg" "$port" "$admin_port"
assert_contains "$(admin_curl "$token" GET /api/organization)" 'Persist Org'
assert_contains "$(admin_curl "$token" GET /api/members)" 'dev@example.com'

"$AMAQUET_CLI" -uri "amaquet://127.0.0.1:$port" -api-key "$secret" PING '{}' >/dev/null
custom_stop_server

rm -rf "$AMAQUET_TEST_TMP"
echo "PASS admin state restart"
