#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:ordered_set:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"ordered_set\",\"options\":{}}" >/dev/null
key_lifecycle "$KEY" ordered_set

cli OP "{\"key\":\"$KEY\",\"operation\":\"ADD\",\"args\":{\"value\":{\"type\":\"utf8_string\",\"value\":\"alpha\"}}}" >/dev/null
cli OP "{\"key\":\"$KEY\",\"operation\":\"ADD\",\"args\":{\"value\":{\"type\":\"utf8_string\",\"value\":\"beta\"}}}" >/dev/null

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"VALUES\",\"args\":{}}")"
assert_contains "$out" alpha "VALUES"

key_del "$KEY"
finish ordered_set
