#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:cbor:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"cbor\",\"options\":{}}" >/dev/null
key_lifecycle "$KEY" cbor

out="$(cli GET "{\"key\":\"$KEY\"}")"
assert_contains "$out" '"type": "cbor"' "GET"

key_del "$KEY"
finish cbor
