#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/custom_server.sh"

AMAQUET_TEST_TMP="$(mktemp -d -t amaquet-aof-compress.XXXXXX)"
port="$(free_port)"
admin_port="$(free_port)"
cfg="$AMAQUET_TEST_TMP/amaquet.json"

cat >"$cfg" <<JSON
{"protocol":{"host":"127.0.0.1","port":$port,"tls":false,"require_auth":false},"admin":{"host":"127.0.0.1","port":$admin_port},"security":{"bootstrap_token":"aof-compress"},"persistence":{"aof_enabled":true,"aof_path":"$AMAQUET_TEST_TMP/db.aof","fsync":"always"},"limits":{"max_connections":64,"max_payload_bytes":8388608},"compression":{"enabled":true,"algorithm":"auto","min_bytes":512,"min_savings_percent":5},"data_dir":"$AMAQUET_TEST_TMP/data"}
JSON

custom_start_server "$cfg" "$port" "$admin_port"
payload_file="$AMAQUET_TEST_TMP/compress-set.json"
python3 >"$payload_file" <<'PY'
import json
print(json.dumps({"key":"compress:persist","type":"utf8_string","value":"ABCD"*40000}))
PY

"$AMAQUET_CLI" -uri "$AMAQUET_URI" SET "@$payload_file" >/dev/null
assert_true "$(amaquet MEMORY '{"key":"compress:persist"}' | python3 -c 'import json,sys;print(json.load(sys.stdin).get("compressed",False))')"
custom_stop_server

custom_start_server "$cfg" "$port" "$admin_port"
assert_eq "$(amaquet GET '{"key":"compress:persist"}' | python3 -c 'import json,sys;print(len(json.load(sys.stdin)["value"]))')" 160000
assert_true "$(amaquet MEMORY '{"key":"compress:persist"}' | python3 -c 'import json,sys;print(json.load(sys.stdin).get("compressed",False))')"
custom_stop_server

rm -rf "$AMAQUET_TEST_TMP"
echo "PASS AOF compression replay"
