#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:ordered_map:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"ordered_map\",\"options\":{}}" >/dev/null
key_lifecycle "$KEY" ordered_map

cli OP "{\"key\":\"$KEY\",\"operation\":\"SET\",\"args\":{\"field\":\"name\",\"value\":{\"type\":\"utf8_string\",\"value\":\"alpha\"}}}" >/dev/null

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"GET\",\"args\":{\"field\":\"name\"}}")"
assert_contains "$out" alpha "GET"

key_del "$KEY"
finish ordered_map
