#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:polygon:$$:$RANDOM"

start_server

cli SET "{\"key\":\"$KEY\",\"type\":\"polygon\",\"value\":[{\"Lat\":14.5,\"Lon\":120.9},{\"Lat\":14.7,\"Lon\":120.9},{\"Lat\":14.6,\"Lon\":121.1}]}" >/dev/null
key_lifecycle "$KEY" polygon

out="$(extract "$(op "$KEY" CONTAINS '{"point":{"Lat":14.6,"Lon":121.0}}')")"
assert_true "$out" "inside polygon"

key_del "$KEY"
finish polygon
