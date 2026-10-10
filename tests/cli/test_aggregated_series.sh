#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:aggregated_series:$$:$RANDOM"
SRC="cli:aggregated_series:src:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"aggregated_series\",\"options\":{\"bucket_ms\":1000,\"aggregation\":\"avg\"}}" >/dev/null
cli CREATE "{\"key\":\"$SRC\",\"type\":\"timeseries\",\"options\":{}}" >/dev/null
key_lifecycle "$KEY" aggregated_series

cli OP "{\"key\":\"$SRC\",\"operation\":\"ADD\",\"args\":{\"timestamp\":1000,\"value\":10}}" >/dev/null
cli OP "{\"key\":\"$SRC\",\"operation\":\"ADD\",\"args\":{\"timestamp\":1500,\"value\":20}}" >/dev/null

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"AGGREGATE\",\"args\":{\"source_key\":\"$SRC\",\"from\":0,\"to\":3000,\"bucket_ms\":1000,\"aggregation\":\"avg\"}}")"
assert_contains "$out" 15 "AGGREGATE avg"
cli DEL "{\"keys\":[\"$SRC\"]}" >/dev/null

key_del "$KEY"
finish aggregated_series
