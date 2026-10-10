#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:bit_set:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"bit_set\",\"options\":{}}" >/dev/null
key_lifecycle "$KEY" bit_set

op "$KEY" SET '{"bit":7,"value":true}' >/dev/null
op "$KEY" SET '{"bit":42,"value":true}' >/dev/null

out="$(extract "$(op "$KEY" GETBIT '{"bit":7}')")"
assert_true "$out" "GETBIT 7"

out="$(extract "$(op "$KEY" COUNT '{}')")"
assert_eq "$out" 2 "COUNT"

key_del "$KEY"
finish bit_set
