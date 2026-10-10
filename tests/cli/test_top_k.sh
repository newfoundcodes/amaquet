#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:top_k:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"top_k\",\"options\":{\"k\":5}}" >/dev/null
key_lifecycle "$KEY" top_k

cli OP "{\"key\":\"$KEY\",\"operation\":\"ADD\",\"args\":{\"value\":\"alice\",\"count\":10}}" >/dev/null
cli OP "{\"key\":\"$KEY\",\"operation\":\"ADD\",\"args\":{\"value\":\"bob\",\"count\":5}}" >/dev/null
cli OP "{\"key\":\"$KEY\",\"operation\":\"ADD\",\"args\":{\"value\":\"carol\",\"count\":8}}" >/dev/null

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"ITEMS\",\"args\":{}}")"
assert_contains "$out" alice "ITEMS"
assert_contains "$out" carol "ITEMS"

key_del "$KEY"
finish top_k
