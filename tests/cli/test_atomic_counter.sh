#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:atomic_counter:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"atomic_counter\",\"options\":{\"initial\":10}}" >/dev/null
key_lifecycle "$KEY" atomic_counter

out="$(extract "$(op "$KEY" ADD '{"delta":5}')")"
assert_eq "$out" 15 "ADD"

out="$(extract "$(op "$KEY" CAS '{"old":15,"new":20}')")"
assert_true "$out" "CAS success"

out="$(extract "$(op "$KEY" LOAD '{}')")"
assert_eq "$out" 20 "LOAD after CAS"

# CAS failure on wrong old value
out="$(extract "$(op "$KEY" CAS '{"old":99,"new":0}')")"
assert_false "$out" "CAS fail"

key_del "$KEY"
finish atomic_counter
