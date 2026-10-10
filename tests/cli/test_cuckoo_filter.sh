#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:cuckoo_filter:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"cuckoo_filter\",\"options\":{\"capacity\":1024}}" >/dev/null
key_lifecycle "$KEY" cuckoo_filter

op "$KEY" ADD '{"value":"alice"}' >/dev/null

out="$(extract "$(op "$KEY" CONTAINS '{"value":"alice"}')")"
assert_true "$out" "CONTAINS"

op "$KEY" REMOVE '{"value":"alice"}' >/dev/null

out="$(extract "$(op "$KEY" CONTAINS '{"value":"alice"}')")"
assert_false "$out" "NOT CONTAINS after REMOVE"

key_del "$KEY"
finish cuckoo_filter
