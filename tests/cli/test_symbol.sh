#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:symbol:$$:$RANDOM"

start_server

cli SET "{\"key\":\"$KEY\",\"type\":\"symbol\",\"value\":\"ACTIVE\"}" >/dev/null
key_lifecycle "$KEY" symbol

out="$(cli GET "{\"key\":\"$KEY\"}")"
assert_contains "$out" "ACTIVE" "GET"

key_del "$KEY"
finish symbol
