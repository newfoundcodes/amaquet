#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/scenario.sh"

scenario_init
assert_command_fails amaquet SET '{}'
assert_command_fails amaquet SET '{"key":"","type":"integer","value":1}'
assert_command_fails amaquet SET '{"key":"bad-type","type":"not_a_type","value":1}'
assert_command_fails amaquet SET '{"key":"bad-int","type":"integer","value":"x"}'
assert_command_fails amaquet SET '{"key":"bad-bin","type":"binary_string","value":"%%%"}'
assert_command_fails amaquet EXPIRE '{"key":"missing","ttl_ms":100}'
assert_command_fails amaquet EXPIRE '{"key":"missing","ttl_ms":0}'
assert_command_fails amaquet GET '{}'
assert_command_fails amaquet DEL '{}'

server_healthy
echo "PASS invalid arguments"
