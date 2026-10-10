#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:utf8_string:$$:$RANDOM"

start_server

cli SET "{\"key\":\"$KEY\",\"type\":\"utf8_string\",\"value\":\"Hello, Amaquet\"}" >/dev/null
key_lifecycle "$KEY" utf8_string

out="$(cli GET "{\"key\":\"$KEY\"}")"
assert_contains "$out" '"type": "utf8_string"' "GET"
assert_contains "$out" "Hello, Amaquet" "value"

key_del "$KEY"
finish utf8_string
