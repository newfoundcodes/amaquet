#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/common.sh"

start_test_server

out="$(amaquet PING '{}' | json_extract 'x')"
assert_eq "$out" PONG

out="$(amaquet INFO '{}')"
assert_contains "$out" Amaquet
assert_contains "$out" protocol_version

out="$(amaquet TYPES '{}')"
assert_contains "$out" utf8_string
assert_contains "$out" hnsw_index

assert_command_fails amaquet NO_SUCH_COMMAND '{}'
assert_command_fails "$AMAQUET_CLI" -uri 'redis://127.0.0.1:6379' PING '{}'

log "PROTOCOL SMOKE PASS"
