#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/custom_server.sh"

need curl

AMAQUET_TEST_TMP="$(mktemp -d -t amaquet-admin-input.XXXXXX)"
port="$(free_port)"
admin_port="$(free_port)"
token='bootstrap-admin-input'

cat >"$AMAQUET_TEST_TMP/amaquet.json" <<JSON
{"protocol":{"host":"127.0.0.1","port":$port,"tls":false,"require_auth":true},"admin":{"host":"127.0.0.1","port":$admin_port},"security":{"bootstrap_token":"$token"},"persistence":{"aof_enabled":false,"aof_path":"$AMAQUET_TEST_TMP/db.aof","fsync":"no"},"limits":{"max_connections":64,"max_payload_bytes":8388608},"compression":{"enabled":true,"algorithm":"auto","min_bytes":1024,"min_savings_percent":8},"data_dir":"$AMAQUET_TEST_TMP/data"}
JSON

custom_start_server "$AMAQUET_TEST_TMP/amaquet.json" "$port" "$admin_port"

[[ "$(curl -sS -o /dev/null -w '%{http_code}' "http://127.0.0.1:$admin_port/api/overview")" == 401 ]] || die "overview allowed without token"
[[ "$(admin_status "$token" POST /api/organization '{}')" == 405 ]] || die "invalid method not rejected"
[[ "$(admin_status "$token" PUT /api/organization '{not-json')" == 400 ]] || die "malformed JSON not rejected"
[[ "$(admin_status "$token" PUT /api/organization '{"name":"A","slug":"a","unknown":1}')" == 400 ]] || die "unknown JSON field not rejected"
[[ "$(admin_status "$token" POST /api/members '{"name":"x","email":"x@example.com","role":"root"}')" == 400 ]] || die "invalid role not rejected"
[[ "$(admin_status "$token" POST /api/data/command '{"command":"NOPE","args":{}}')" == 400 ]] || die "unknown command not rejected"

for i in $(seq 1 25); do
    code="$(curl -sS -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $token" -H 'Content-Type: application/json' -X POST -d "{\"command\":\"PING\",\"args\":{},\"junk$i\":true}" "http://127.0.0.1:$admin_port/api/data/command")"
    [[ "$code" == 200 || "$code" == 400 ]] || die "unexpected HTTP code $code"
done

"$AMAQUET_CLI" -uri "amaquet://127.0.0.1:$port" -api-key "$token" PING '{}' >/dev/null
custom_stop_server

rm -rf "$AMAQUET_TEST_TMP"
echo "PASS admin API input fuzz"
