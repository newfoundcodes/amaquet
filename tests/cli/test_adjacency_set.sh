#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:adjacency_set:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"adjacency_set\",\"options\":{}}" >/dev/null
key_lifecycle "$KEY" adjacency_set

cli OP "{\"key\":\"$KEY\",\"operation\":\"ADD\",\"args\":{\"from\":\"a\",\"to\":\"b\",\"weight\":1.5}}" >/dev/null
cli OP "{\"key\":\"$KEY\",\"operation\":\"ADD\",\"args\":{\"from\":\"a\",\"to\":\"c\",\"weight\":2.0}}" >/dev/null
cli OP "{\"key\":\"$KEY\",\"operation\":\"ADD\",\"args\":{\"from\":\"b\",\"to\":\"d\",\"weight\":0.5}}" >/dev/null

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"NEIGHBORS\",\"args\":{\"from\":\"a\"}}")"
assert_contains "$out" b "NEIGHBORS of a"
assert_contains "$out" c "NEIGHBORS of a"

key_del "$KEY"
finish adjacency_set
