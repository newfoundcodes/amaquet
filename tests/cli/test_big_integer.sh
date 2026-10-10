#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:big_integer:$$:$RANDOM"

start_server

cli SET "{\"key\":\"$KEY\",\"type\":\"big_integer\",\"value\":\"1234567890123456789012345678901234567890\"}" >/dev/null
key_lifecycle "$KEY" big_integer

out="$(extract "$(op "$KEY" ADD '{"value":"10"}')")"
assert_contains "$out" "1234567890123456789012345678901234567900" "ADD"

key_del "$KEY"
finish big_integer
