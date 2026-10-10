#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:cidr_network:$$:$RANDOM"

start_server

cli SET "{\"key\":\"$KEY\",\"type\":\"cidr_network\",\"value\":\"10.0.0.0/8\"}" >/dev/null
key_lifecycle "$KEY" cidr_network

out="$(extract "$(op "$KEY" CONTAINS '{"ip":"10.1.2.3"}')")"
assert_true "$out" "CONTAINS"

out="$(extract "$(op "$KEY" CONTAINS '{"ip":"192.168.1.1"}')")"
assert_false "$out" "NOT_CONTAINS"

key_del "$KEY"
finish cidr_network
