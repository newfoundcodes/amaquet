#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:multimap:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"multimap\",\"options\":{}}" >/dev/null
key_lifecycle "$KEY" multimap

cli OP "{\"key\":\"$KEY\",\"operation\":\"ADD\",\"args\":{\"field\":\"tag\",\"value\":{\"type\":\"utf8_string\",\"value\":\"alpha\"}}}" >/dev/null
cli OP "{\"key\":\"$KEY\",\"operation\":\"ADD\",\"args\":{\"field\":\"tag\",\"value\":{\"type\":\"utf8_string\",\"value\":\"beta\"}}}" >/dev/null

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"GET\",\"args\":{\"field\":\"tag\"}}")"
assert_contains "$out" alpha "GET"
assert_contains "$out" beta "GET multi"

key_del "$KEY"
finish multimap
