#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:event_log:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"event_log\",\"options\":{}}" >/dev/null
key_lifecycle "$KEY" event_log

op "$KEY" APPEND '{"event_type":"created","aggregate_id":"agg1","payload":{"type":"utf8_string","value":"born"}}' >/dev/null
op "$KEY" APPEND '{"event_type":"updated","aggregate_id":"agg1","payload":{"type":"utf8_string","value":"changed"}}' >/dev/null

out="$(extract "$(op "$KEY" LEN '{}')")"
assert_eq "$out" 2 "LEN"

key_del "$KEY"
finish event_log
