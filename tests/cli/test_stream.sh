#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:stream:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"stream\",\"options\":{}}" >/dev/null
key_lifecycle "$KEY" stream

cli OP "{\"key\":\"$KEY\",\"operation\":\"ADD\",\"args\":{\"fields\":{\"msg\":{\"type\":\"utf8_string\",\"value\":\"hello\"}}}}" >/dev/null
cli OP "{\"key\":\"$KEY\",\"operation\":\"ADD\",\"args\":{\"fields\":{\"msg\":{\"type\":\"utf8_string\",\"value\":\"world\"}}}}" >/dev/null

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"RANGE\",\"args\":{\"count\":10}}")"
assert_contains "$out" hello "RANGE"
assert_contains "$out" world "RANGE"

cli OP "{\"key\":\"$KEY\",\"operation\":\"GROUP_CREATE\",\"args\":{\"group\":\"g1\"}}" >/dev/null

key_del "$KEY"
finish stream
