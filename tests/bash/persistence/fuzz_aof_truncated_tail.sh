#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/custom_server.sh"
AMAQUET_TEST_TMP="$(mktemp -d -t amaquet-aof-tail.XXXXXX)"

port="$(free_port)"
admin_port="$(free_port)"
cfg="$AMAQUET_TEST_TMP/amaquet.json"
aof="$AMAQUET_TEST_TMP/db.aof"

cat >"$cfg" <<JSON
{"protocol":{"host":"127.0.0.1","port":$port,"tls":false,"require_auth":false},"admin":{"host":"127.0.0.1","port":$admin_port},"security":{"bootstrap_token":"aof-tail"},"persistence":{"aof_enabled":true,"aof_path":"$aof","fsync":"always"},"limits":{"max_connections":64,"max_payload_bytes":8388608},"compression":{"enabled":true,"algorithm":"auto","min_bytes":1024,"min_savings_percent":8},"data_dir":"$AMAQUET_TEST_TMP/data"}
JSON

custom_start_server "$cfg" "$port" "$admin_port"
setv tail:first integer 1
setv tail:second utf8_string '"this record will be truncated"'

custom_stop_server
size=$(stat -c %s "$aof" 2>/dev/null || stat -f %z "$aof")
((size > 32)) || die "AOF unexpectedly small"

python3 - "$aof" <<'PY'
import os,sys
p=sys.argv[1]; n=os.path.getsize(p); os.truncate(p,max(10,n-12))
PY

custom_start_server "$cfg" "$port" "$admin_port"
assert_contains "$(amaquet GET '{"key":"tail:first"}')" '"value": 1'
amaquet PING '{}' >/dev/null
custom_stop_server

rm -rf "$AMAQUET_TEST_TMP"
echo "PASS truncated AOF tail"
