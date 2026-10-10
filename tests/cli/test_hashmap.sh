#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:hashmap:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"hashmap\",\"options\":{}}" >/dev/null
key_lifecycle "$KEY" hashmap

cli OP "{\"key\":\"$KEY\",\"operation\":\"SET\",\"args\":{\"field\":\"name\",\"value\":{\"type\":\"utf8_string\",\"value\":\"alpha\"}}}" >/dev/null
cli OP "{\"key\":\"$KEY\",\"operation\":\"SET\",\"args\":{\"field\":\"count\",\"value\":{\"type\":\"integer\",\"value\":99}}}" >/dev/null

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"GET\",\"args\":{\"field\":\"name\"}}")"
assert_contains "$out" alpha "GET name"

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"GET\",\"args\":{\"field\":\"count\"}}")"
assert_contains "$out" 99 "GET count"

key_del "$KEY"
finish hashmap
