#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:vector_set:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"vector_set\",\"options\":{\"dim\":3,\"metric\":\"cosine\"}}" >/dev/null
key_lifecycle "$KEY" vector_set

cli OP "{\"key\":\"$KEY\",\"operation\":\"ADD\",\"args\":{\"id\":\"v1\",\"vector\":[1,0,0],\"metadata\":{\"class\":\"a\"}}}" >/dev/null
cli OP "{\"key\":\"$KEY\",\"operation\":\"ADD\",\"args\":{\"id\":\"v2\",\"vector\":[0,1,0],\"metadata\":{\"class\":\"b\"}}}" >/dev/null

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"SEARCH\",\"args\":{\"vector\":[1,0,0],\"k\":1}}")"
assert_contains "$out" v1 "SEARCH nearest"

key_del "$KEY"
finish vector_set
