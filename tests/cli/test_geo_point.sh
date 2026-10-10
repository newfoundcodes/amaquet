#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:geo_point:$$:$RANDOM"

start_server

cli SET "{\"key\":\"$KEY\",\"type\":\"geo_point\",\"value\":{\"Lat\":14.5995,\"Lon\":120.9842}}" >/dev/null
key_lifecycle "$KEY" geo_point
out="$(extract "$(op "$KEY" DISTANCE '{"point":{"Lat":14.5995,"Lon":120.9842}}')")"
assert_eq "$out" 0 "self-distance"
key_del "$KEY"
finish geo_point
