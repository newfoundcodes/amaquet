#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/custom_server.sh"

AMAQUET_TEST_TMP="$(mktemp -d -t amaquet-aof-replay.XXXXXX)"
port="$(free_port)"
admin_port="$(free_port)"
cfg="$AMAQUET_TEST_TMP/amaquet.json"

cat >"$cfg" <<JSON
{"protocol":{"host":"127.0.0.1","port":$port,"tls":false,"require_auth":false},"admin":{"host":"127.0.0.1","port":$admin_port},"security":{"bootstrap_token":"aof-test"},"persistence":{"aof_enabled":true,"aof_path":"$AMAQUET_TEST_TMP/db.aof","fsync":"always"},"limits":{"max_connections":64,"max_payload_bytes":8388608},"compression":{"enabled":true,"algorithm":"auto","min_bytes":1024,"min_savings_percent":8},"data_dir":"$AMAQUET_TEST_TMP/data"}
JSON

custom_start_server "$cfg" "$port" "$admin_port"
setv persist:int integer 42

create persist:list list '{}'
op persist:list PUSH_BACK '{"value":{"type":"utf8_string","value":"hello"}}' >/dev/null

create persist:topic persistent_topic '{"max_messages":100}'
op persist:topic PUBLISH '{"message_key":"m1","value":{"type":"integer","value":7}}' >/dev/null

custom_stop_server
custom_start_server "$cfg" "$port" "$admin_port"

assert_contains "$(amaquet GET '{"key":"persist:int"}')" '"value": 42'
assert_contains "$(op persist:list RANGE '{"start":0,"stop":-1}')" hello
assert_contains "$(op persist:topic READ '{"from":0,"count":10}')" '"value": 7'

custom_stop_server
rm -rf "$AMAQUET_TEST_TMP"

echo "PASS AOF restart/replay"
