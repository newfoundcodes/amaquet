#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:array:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"array\",\"options\":{\"typed\":true,\"element_type\":\"utf8_string\"}}" >/dev/null
key_lifecycle "$KEY" array

cli OP "{\"key\":\"$KEY\",\"operation\":\"APPEND\",\"args\":{\"value\":{\"type\":\"utf8_string\",\"value\":\"alpha\"}}}" >/dev/null
cli OP "{\"key\":\"$KEY\",\"operation\":\"APPEND\",\"args\":{\"value\":{\"type\":\"utf8_string\",\"value\":\"beta\"}}}" >/dev/null

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"GET\",\"args\":{\"index\":0}}")"
assert_contains "$out" alpha "GET[0]"

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"GET\",\"args\":{\"index\":1}}")"
assert_contains "$out" beta "GET[1]"

key_del "$KEY"
finish array
