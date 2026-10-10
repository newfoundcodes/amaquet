#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:binary_string:$$:$RANDOM"

start_server

cli SET "{\"key\":\"$KEY\",\"type\":\"binary_string\",\"value\":\"AAECaGVsbG8=\"}" >/dev/null
key_lifecycle "$KEY" binary_string

out="$(cli GET "{\"key\":\"$KEY\"}")"
assert_contains "$out" '"type": "binary_string"' "GET"

key_del "$KEY"
finish binary_string
