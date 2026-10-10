#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:vector_metadata:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"vector_metadata\",\"options\":{}}" >/dev/null
key_lifecycle "$KEY" vector_metadata

cli OP "{\"key\":\"$KEY\",\"operation\":\"SET\",\"args\":{\"field\":\"lang\",\"value\":\"en\"}}" >/dev/null
cli OP "{\"key\":\"$KEY\",\"operation\":\"SET\",\"args\":{\"field\":\"model\",\"value\":\"bert\"}}" >/dev/null

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"GET\",\"args\":{}}")"
assert_contains "$out" en "GET lang"
assert_contains "$out" bert "GET model"

key_del "$KEY"
finish vector_metadata
