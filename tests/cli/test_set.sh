#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:set:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"set\",\"options\":{}}" >/dev/null
key_lifecycle "$KEY" set

out="$(extract "$(op "$KEY" ADD '{"value":{"type":"utf8_string","value":"alpha"}}')")"
assert_true "$out" "ADD"

out="$(extract "$(op "$KEY" CONTAINS '{"value":{"type":"utf8_string","value":"alpha"}}')")"
assert_true "$out" "CONTAINS"

out="$(extract "$(op "$KEY" ADD '{"value":{"type":"utf8_string","value":"alpha"}}')")"
assert_false "$out" "DUPLICATE"

key_del "$KEY"
finish set
