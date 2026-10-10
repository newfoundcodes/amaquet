#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:timestamp:$$:$RANDOM"

start_server

cli SET "{\"key\":\"$KEY\",\"type\":\"timestamp\",\"value\":\"2026-10-01T00:00:00Z\"}" >/dev/null
key_lifecycle "$KEY" timestamp

out="$(cli GET "{\"key\":\"$KEY\"}")"
assert_contains "$out" '"type": "timestamp"' "GET"

key_del "$KEY"
finish timestamp
