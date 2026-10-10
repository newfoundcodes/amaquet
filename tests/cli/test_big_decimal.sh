#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:big_decimal:$$:$RANDOM"

start_server

cli SET "{\"key\":\"$KEY\",\"type\":\"big_decimal\",\"value\":\"12345678901234567890.000000000000000001\"}" >/dev/null
key_lifecycle "$KEY" big_decimal

op "$KEY" ADD '{"value":"0.3211"}' >/dev/null

out="$(extract "$(op "$KEY" CMP '{"value":"12345678901234567890.321100000000000001"}')")"
assert_eq "$out" 0 "CMP"

key_del "$KEY"
finish big_decimal
