#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:uuid:$$:$RANDOM"

start_server

cli SET "{\"key\":\"$KEY\",\"type\":\"uuid\",\"value\":\"550e8400-e29b-41d4-a716-446655440000\"}" >/dev/null
key_lifecycle "$KEY" uuid

out="$(cli GET "{\"key\":\"$KEY\"}")"
assert_contains "$out" "550e8400" "GET"

key_del "$KEY"
finish uuid
