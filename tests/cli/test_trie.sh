#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:trie:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"trie\",\"options\":{}}" >/dev/null
key_lifecycle "$KEY" trie

cli OP "{\"key\":\"$KEY\",\"operation\":\"INSERT\",\"args\":{\"value\":\"amaquet\",\"id\":\"1\"}}" >/dev/null
cli OP "{\"key\":\"$KEY\",\"operation\":\"INSERT\",\"args\":{\"value\":\"ama\",\"id\":\"2\"}}" >/dev/null

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"PREFIX\",\"args\":{\"prefix\":\"ama\",\"limit\":10}}")"
assert_contains "$out" 1 "PREFIX"
assert_contains "$out" 2 "PREFIX 2"

key_del "$KEY"
finish trie
