#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:blocking_queue:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"blocking_queue\",\"options\":{}}" >/dev/null
key_lifecycle "$KEY" blocking_queue

cli OP "{\"key\":\"$KEY\",\"operation\":\"ENQUEUE\",\"args\":{\"value\":{\"type\":\"utf8_string\",\"value\":\"alpha\"}}}" >/dev/null

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"DEQUEUE\",\"args\":{}}")"
assert_contains "$out" alpha "DEQUEUE"

key_del "$KEY"
finish blocking_queue
