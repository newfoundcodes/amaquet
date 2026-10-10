#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/custom_server.sh"
need curl

AMAQUET_TEST_TMP="$(mktemp -d -t amaquet-auth.XXXXXX)"
port="$(free_port)"
admin_port="$(free_port)"
token='bootstrap-auth-fuzz'

cat >"$AMAQUET_TEST_TMP/amaquet.json" <<JSON
{"protocol":{"host":"127.0.0.1","port":$port,"tls":false,"require_auth":true},"admin":{"host":"127.0.0.1","port":$admin_port},"security":{"bootstrap_token":"$token"},"persistence":{"aof_enabled":false,"aof_path":"$AMAQUET_TEST_TMP/db.aof","fsync":"no"},"limits":{"max_connections":128,"max_payload_bytes":8388608},"compression":{"enabled":true,"algorithm":"auto","min_bytes":1024,"min_savings_percent":8},"data_dir":"$AMAQUET_TEST_TMP/data"}
JSON

custom_start_server "$AMAQUET_TEST_TMP/amaquet.json" "$port" "$admin_port"

# No authentication must fail.
if "$AMAQUET_CLI" -uri "amaquet://127.0.0.1:$port" PING '{}' >/dev/null 2>&1; then die "unauthenticated protocol request succeeded"; fi

admin_uri="amaquet://127.0.0.1:$port"
"$AMAQUET_CLI" -uri "$admin_uri" -api-key "$token" PING '{}' >/dev/null

# Create developer and auditor credentials through the admin API.
dev_json="$(admin_curl "$token" POST /api/api-keys '{"name":"dev-fuzz","role":"developer"}')"
dev="$(printf '%s' "$dev_json" | python3 -c 'import json,sys;print(json.load(sys.stdin)["secret"])')"
dev_id="$(printf '%s' "$dev_json" | python3 -c 'import json,sys;print(json.load(sys.stdin)["api_key"]["id"])')"
aud_json="$(admin_curl "$token" POST /api/api-keys '{"name":"aud-fuzz","role":"auditor"}')"
aud="$(printf '%s' "$aud_json" | python3 -c 'import json,sys;print(json.load(sys.stdin)["secret"])')"
dev_uri="amaquet://127.0.0.1:$port"
aud_uri="amaquet://127.0.0.1:$port"

"$AMAQUET_CLI" -uri "$dev_uri" -api-key "$dev" SET '{"key":"rbac:key","type":"integer","value":1}' >/dev/null
"$AMAQUET_CLI" -uri "$aud_uri" -api-key "$aud" GET '{"key":"rbac:key"}' >/dev/null

if "$AMAQUET_CLI" -uri "$aud_uri" -api-key "$aud" SET '{"key":"rbac:no","type":"integer","value":2}' >/dev/null 2>&1; then die "auditor performed data.write"; fi

[[ "$(admin_status "$aud" GET /api/audit)" == 200 ]] || die "auditor should read audit"
[[ "$(admin_status "$aud" POST /api/api-keys '{"name":"bad","role":"developer"}')" == 403 ]] || die "auditor created API key"
[[ "$(admin_status "$dev" GET /api/organization)" == 200 ]] || die "developer should read organization"
[[ "$(admin_status "$dev" PUT /api/organization '{"name":"x","slug":"x"}')" == 403 ]] || die "developer changed organization"

admin_curl "$token" DELETE "/api/api-keys/$dev_id" >/dev/null

if "$AMAQUET_CLI" -uri "$dev_uri" -api-key "$dev" PING '{}' >/dev/null 2>&1; then die "revoked API key still authenticated on new connection"; fi

"$AMAQUET_CLI" -uri "$admin_uri" -api-key "$token" PING '{}' >/dev/null
custom_stop_server

rm -rf "$AMAQUET_TEST_TMP"
echo "PASS auth/RBAC fuzz"
