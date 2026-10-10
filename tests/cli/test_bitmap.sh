#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:bitmap:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"bitmap\",\"options\":{}}" >/dev/null
key_lifecycle "$KEY" bitmap

op "$KEY" SET '{"bit":7,"value":true}' >/dev/null
op "$KEY" SET '{"bit":15,"value":true}' >/dev/null

out="$(extract "$(op "$KEY" GETBIT '{"bit":7}')")"
assert_true "$out" "GETBIT 7"

out="$(extract "$(op "$KEY" GETBIT '{"bit":0}')")"
assert_false "$out" "GETBIT 0"

out="$(extract "$(op "$KEY" COUNT '{}')")"
assert_eq "$out" 2 "COUNT"

key_del "$KEY"
finish bitmap
