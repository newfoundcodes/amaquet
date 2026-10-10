#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:sorted_set:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"sorted_set\",\"options\":{}}" >/dev/null
key_lifecycle "$KEY" sorted_set

op "$KEY" ADD '{"member":{"type":"utf8_string","value":"alpha"},"score":10.5}' >/dev/null
op "$KEY" ADD '{"member":{"type":"utf8_string","value":"beta"},"score":5.0}' >/dev/null

out="$(extract "$(op "$KEY" SCORE '{"member":{"type":"utf8_string","value":"alpha"}}')")"
assert_eq "$out" 10.5 "SCORE"

key_del "$KEY"
finish sorted_set
