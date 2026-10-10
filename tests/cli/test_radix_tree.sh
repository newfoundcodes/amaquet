#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:radix_tree:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"radix_tree\",\"options\":{}}" >/dev/null
key_lifecycle "$KEY" radix_tree

cli OP "{\"key\":\"$KEY\",\"operation\":\"INSERT\",\"args\":{\"value\":\"amaquet\",\"id\":\"1\"}}" >/dev/null
cli OP "{\"key\":\"$KEY\",\"operation\":\"INSERT\",\"args\":{\"value\":\"amazon\",\"id\":\"2\"}}" >/dev/null
cli OP "{\"key\":\"$KEY\",\"operation\":\"INSERT\",\"args\":{\"value\":\"banana\",\"id\":\"3\"}}" >/dev/null

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"PREFIX\",\"args\":{\"prefix\":\"ama\",\"limit\":10}}")"
assert_contains "$out" 1 "PREFIX ama id1"
assert_contains "$out" 2 "PREFIX ama id2"

key_del "$KEY"
finish radix_tree
