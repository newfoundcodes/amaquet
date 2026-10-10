#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:hash_index:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"hash_index\",\"options\":{}}" >/dev/null
key_lifecycle "$KEY" hash_index

cli OP "{\"key\":\"$KEY\",\"operation\":\"ADD\",\"args\":{\"value\":\"blue\",\"id\":\"1\"}}" >/dev/null
cli OP "{\"key\":\"$KEY\",\"operation\":\"ADD\",\"args\":{\"value\":\"blue\",\"id\":\"2\"}}" >/dev/null
cli OP "{\"key\":\"$KEY\",\"operation\":\"ADD\",\"args\":{\"value\":\"red\",\"id\":\"3\"}}" >/dev/null

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"LOOKUP\",\"args\":{\"value\":\"blue\"}}")"
assert_contains "$out" 1 "LOOKUP blue"
assert_contains "$out" 2 "LOOKUP blue multi"

key_del "$KEY"
finish hash_index
