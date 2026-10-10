#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:roaring_bitmap:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"roaring_bitmap\",\"options\":{}}" >/dev/null
key_lifecycle "$KEY" roaring_bitmap

op "$KEY" ADD '{"value":42}' >/dev/null
op "$KEY" ADD '{"value":1000000}' >/dev/null

out="$(extract "$(op "$KEY" CONTAINS '{"value":42}')")"
assert_true "$out" "CONTAINS 42"

out="$(extract "$(op "$KEY" CONTAINS '{"value":7}')")"
assert_false "$out" "NOT CONTAINS 7"

out="$(extract "$(op "$KEY" CARDINALITY '{}')")"
assert_eq "$out" 2 "CARDINALITY"

key_del "$KEY"
finish roaring_bitmap
