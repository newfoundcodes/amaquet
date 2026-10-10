#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:url:$$:$RANDOM"

start_server

cli SET "{\"key\":\"$KEY\",\"type\":\"url\",\"value\":\"https://example.com:8443/path?q=amaquet\"}" >/dev/null
key_lifecycle "$KEY" url

out="$(extract "$(op "$KEY" HOST '{}')")"
assert_contains "$out" example.com "HOST"

out="$(extract "$(op "$KEY" SCHEME '{}')")"
assert_eq "$out" https "SCHEME"

key_del "$KEY"
finish url
