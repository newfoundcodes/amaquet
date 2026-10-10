#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:deque:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"deque\",\"options\":{\"capacity\":8}}" >/dev/null
key_lifecycle "$KEY" deque

op "$KEY" PUSH_BACK '{"value":{"type":"utf8_string","value":"alpha"}}' >/dev/null
op "$KEY" PUSH_FRONT '{"value":{"type":"integer","value":7}}' >/dev/null

out="$(extract "$(op "$KEY" LEN '{}')")"
assert_eq "$out" 2 "LEN"

key_del "$KEY"
finish deque
