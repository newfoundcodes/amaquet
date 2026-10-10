#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:multiset:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"multiset\",\"options\":{}}" >/dev/null
key_lifecycle "$KEY" multiset

op "$KEY" ADD '{"value":{"type":"utf8_string","value":"alpha"},"count":3}' >/dev/null

out="$(extract "$(op "$KEY" COUNT '{"value":{"type":"utf8_string","value":"alpha"}}')")"
assert_eq "$out" 3 "COUNT"

key_del "$KEY"
finish multiset
