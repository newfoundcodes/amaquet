#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:bit_field:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"bit_field\",\"options\":{\"size_bytes\":8}}" >/dev/null
key_lifecycle "$KEY" bit_field

op "$KEY" SET '{"offset":4,"width":8,"value":171}' >/dev/null

out="$(extract "$(op "$KEY" GET '{"offset":4,"width":8}')")"
assert_eq "$out" 171 "GET"

op "$KEY" SET '{"offset":0,"width":4,"value":5}' >/dev/null

out="$(extract "$(op "$KEY" GET '{"offset":0,"width":4}')")"
assert_eq "$out" 5 "GET second field"

key_del "$KEY"
finish bit_field
