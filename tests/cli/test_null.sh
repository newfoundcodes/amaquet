#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:null:$$:$RANDOM"

start_server

cli SET "{\"key\":\"$KEY\",\"type\":\"null\",\"value\":null}" >/dev/null
key_lifecycle "$KEY" null

out="$(cli GET "{\"key\":\"$KEY\"}")"
assert_contains "$out" '"type": "null"' "GET"

key_del "$KEY"
finish null
