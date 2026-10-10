#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:persistent_topic:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"persistent_topic\",\"options\":{\"max_messages\":100}}" >/dev/null
key_lifecycle "$KEY" persistent_topic

cli OP "{\"key\":\"$KEY\",\"operation\":\"PUBLISH\",\"args\":{\"message_key\":\"m1\",\"value\":{\"type\":\"utf8_string\",\"value\":\"hello\"}}}" >/dev/null
cli OP "{\"key\":\"$KEY\",\"operation\":\"PUBLISH\",\"args\":{\"message_key\":\"m2\",\"value\":{\"type\":\"utf8_string\",\"value\":\"world\"}}}" >/dev/null

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"READ\",\"args\":{\"from\":0,\"count\":10}}")"
assert_contains "$out" hello "READ"
assert_contains "$out" world "READ"

key_del "$KEY"
finish persistent_topic
