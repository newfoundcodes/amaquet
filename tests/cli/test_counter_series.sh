#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:counter_series:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"counter_series\",\"options\":{\"labels\":{\"metric\":\"requests\"}}}" >/dev/null
key_lifecycle "$KEY" counter_series

cli OP "{\"key\":\"$KEY\",\"operation\":\"ADD\",\"args\":{\"timestamp\":1000,\"value\":5}}" >/dev/null
cli OP "{\"key\":\"$KEY\",\"operation\":\"ADD\",\"args\":{\"timestamp\":1000,\"value\":7}}" >/dev/null

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"LAST\",\"args\":{}}")"
assert_contains "$out" 12 "LAST value"

key_del "$KEY"
finish counter_series
