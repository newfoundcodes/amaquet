#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:btree_index:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"btree_index\",\"options\":{\"degree\":8}}" >/dev/null
key_lifecycle "$KEY" btree_index

cli OP "{\"key\":\"$KEY\",\"operation\":\"INSERT\",\"args\":{\"value\":\"b\",\"id\":\"2\"}}" >/dev/null
cli OP "{\"key\":\"$KEY\",\"operation\":\"INSERT\",\"args\":{\"value\":\"a\",\"id\":\"1\"}}" >/dev/null
cli OP "{\"key\":\"$KEY\",\"operation\":\"INSERT\",\"args\":{\"value\":\"c\",\"id\":\"3\"}}" >/dev/null

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"SEARCH\",\"args\":{\"value\":\"a\"}}")"
assert_contains "$out" 1 "SEARCH a"

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"SEARCH\",\"args\":{\"value\":\"c\"}}")"
assert_contains "$out" 3 "SEARCH c"

key_del "$KEY"
finish btree_index
