#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:count_min_sketch:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"count_min_sketch\",\"options\":{\"width\":256,\"depth\":4}}" >/dev/null
key_lifecycle "$KEY" count_min_sketch

op "$KEY" ADD '{"value":"alice","count":5}' >/dev/null
op "$KEY" ADD '{"value":"alice","count":3}' >/dev/null

out="$(extract "$(op "$KEY" ESTIMATE '{"value":"alice"}')")"
assert_ge "$out" 8 "ESTIMATE >= 8"

key_del "$KEY"
finish count_min_sketch
