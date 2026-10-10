#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:timeseries_labels:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"timeseries_labels\",\"options\":{\"labels\":{\"env\":\"test\",\"region\":\"local\"}}}" >/dev/null
key_lifecycle "$KEY" timeseries_labels

cli OP "{\"key\":\"$KEY\",\"operation\":\"SET\",\"args\":{\"label\":\"region\",\"value\":\"ph\"}}" >/dev/null

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"GET\",\"args\":{}}")"
assert_contains "$out" ph "GET label"

key_del "$KEY"
finish timeseries_labels
