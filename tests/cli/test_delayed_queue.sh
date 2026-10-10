#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:delayed_queue:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"delayed_queue\",\"options\":{}}" >/dev/null
key_lifecycle "$KEY" delayed_queue

cli OP "{\"key\":\"$KEY\",\"operation\":\"ENQUEUE\",\"args\":{\"id\":\"job1\",\"value\":{\"type\":\"utf8_string\",\"value\":\"payload\"},\"available_at_ms\":0}}" >/dev/null

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"POP_READY\",\"args\":{}}")"
assert_contains "$out" job1 "POP_READY"

key_del "$KEY"
finish delayed_queue
