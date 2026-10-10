#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:ring_buffer:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"ring_buffer\",\"options\":{\"capacity\":4}}" >/dev/null
key_lifecycle "$KEY" ring_buffer

for v in a b c d e; do
    op "$KEY" PUSH '{"value":{"type":"utf8_string","value":"'"$v"'"}}' >/dev/null
done

out="$(extract "$(op "$KEY" LEN '{}')")"
assert_eq "$out" 4 "LEN (ring wrap)"

key_del "$KEY"
finish ring_buffer
