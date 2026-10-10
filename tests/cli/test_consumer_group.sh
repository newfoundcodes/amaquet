#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:consumer_group:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"consumer_group\",\"options\":{\"name\":\"workers\"}}" >/dev/null
key_lifecycle "$KEY" consumer_group

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"SNAPSHOT\",\"args\":{}}")"
assert_contains "$out" workers "SNAPSHOT"

key_del "$KEY"
finish consumer_group
