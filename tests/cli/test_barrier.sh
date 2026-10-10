#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:barrier:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"barrier\",\"options\":{\"target\":1}}" >/dev/null
key_lifecycle "$KEY" barrier

out="$(extract "$(op "$KEY" WAIT '{}')")"
assert_true "$out" "WAIT (target=1 met by creation)"

key_del "$KEY"
finish barrier
