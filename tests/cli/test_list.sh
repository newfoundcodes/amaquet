#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:list:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"list\",\"options\":{}}" >/dev/null
key_lifecycle "$KEY" list

op "$KEY" PUSH_BACK '{"value":{"type":"utf8_string","value":"alpha"}}' >/dev/null
op "$KEY" PUSH_FRONT '{"value":{"type":"integer","value":7}}' >/dev/null

out="$(extract "$(op "$KEY" LEN '{}')")"
assert_eq "$out" 2 "LEN"

out="$(op "$KEY" RANGE '{"start":0,"stop":-1}')"
assert_contains "$out" alpha "RANGE"

key_del "$KEY"
finish list
