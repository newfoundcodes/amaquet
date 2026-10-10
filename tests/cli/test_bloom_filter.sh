#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:bloom_filter:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"bloom_filter\",\"options\":{\"expected\":1000,\"false_positive_rate\":0.01}}" >/dev/null
key_lifecycle "$KEY" bloom_filter

op "$KEY" ADD '{"value":"alice"}' >/dev/null
op "$KEY" ADD '{"value":"bob"}' >/dev/null

out="$(extract "$(op "$KEY" CONTAINS '{"value":"alice"}')")"
assert_true "$out" "CONTAINS alice"

out="$(extract "$(op "$KEY" CONTAINS '{"value":"bob"}')")"
assert_true "$out" "CONTAINS bob"

key_del "$KEY"
finish bloom_filter
