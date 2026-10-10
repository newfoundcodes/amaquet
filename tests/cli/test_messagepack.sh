#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:messagepack:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"messagepack\",\"options\":{}}" >/dev/null
key_lifecycle "$KEY" messagepack

out="$(cli GET "{\"key\":\"$KEY\"}")"
assert_contains "$out" '"type": "messagepack"' "GET"

key_del "$KEY"
finish messagepack
