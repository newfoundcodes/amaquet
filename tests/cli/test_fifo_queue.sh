#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:fifo_queue:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"fifo_queue\",\"options\":{}}" >/dev/null
key_lifecycle "$KEY" fifo_queue

cli OP "{\"key\":\"$KEY\",\"operation\":\"ENQUEUE\",\"args\":{\"value\":{\"type\":\"utf8_string\",\"value\":\"first\"}}}" >/dev/null
cli OP "{\"key\":\"$KEY\",\"operation\":\"ENQUEUE\",\"args\":{\"value\":{\"type\":\"utf8_string\",\"value\":\"second\"}}}" >/dev/null

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"PEEK\",\"args\":{}}")"
assert_contains "$out" first "PEEK"

cli OP "{\"key\":\"$KEY\",\"operation\":\"DEQUEUE\",\"args\":{}}" >/dev/null

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"PEEK\",\"args\":{}}")"
assert_contains "$out" second "PEEK after DEQUEUE"

key_del "$KEY"
finish fifo_queue
