#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:lifo_stack:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"lifo_stack\",\"options\":{}}" >/dev/null
key_lifecycle "$KEY" lifo_stack

cli OP "{\"key\":\"$KEY\",\"operation\":\"PUSH\",\"args\":{\"value\":{\"type\":\"utf8_string\",\"value\":\"first\"}}}" >/dev/null
cli OP "{\"key\":\"$KEY\",\"operation\":\"PUSH\",\"args\":{\"value\":{\"type\":\"utf8_string\",\"value\":\"top\"}}}" >/dev/null

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"PEEK\",\"args\":{}}")"
assert_contains "$out" top "PEEK"

cli OP "{\"key\":\"$KEY\",\"operation\":\"POP\",\"args\":{}}" >/dev/null

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"PEEK\",\"args\":{}}")"
assert_contains "$out" first "PEEK after POP"

key_del "$KEY"
finish lifo_stack
