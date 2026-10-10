#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:decimal:$$:$RANDOM"

start_server

cli SET "{\"key\":\"$KEY\",\"type\":\"decimal\",\"value\":\"12345.6789\"}" >/dev/null
key_lifecycle "$KEY" decimal

op "$KEY" ADD '{"value":"0.3211"}' >/dev/null

out="$(extract "$(op "$KEY" CMP '{"value":"12346.0000"}')")"
assert_eq "$out" 0 "CMP"

key_del "$KEY"
finish decimal
