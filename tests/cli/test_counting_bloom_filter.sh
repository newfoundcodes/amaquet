#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:counting_bloom_filter:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"counting_bloom_filter\",\"options\":{\"expected\":1000,\"false_positive_rate\":0.01}}" >/dev/null
key_lifecycle "$KEY" counting_bloom_filter

op "$KEY" ADD '{"value":"alice"}' >/dev/null

out="$(extract "$(op "$KEY" CONTAINS '{"value":"alice"}')")"
assert_true "$out" "CONTAINS"

op "$KEY" REMOVE '{"value":"alice"}' >/dev/null

key_del "$KEY"
finish counting_bloom_filter
