#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:bounding_box:$$:$RANDOM"

start_server

cli SET "{\"key\":\"$KEY\",\"type\":\"bounding_box\",\"value\":{\"MinLat\":14.0,\"MinLon\":120.0,\"MaxLat\":15.0,\"MaxLon\":122.0}}" >/dev/null
key_lifecycle "$KEY" bounding_box

out="$(extract "$(op "$KEY" CONTAINS '{"point":{"Lat":14.5,"Lon":121.0}}')")"
assert_true "$out" "inside"

out="$(extract "$(op "$KEY" CONTAINS '{"point":{"Lat":16.0,"Lon":121.0}}')")"
assert_false "$out" "outside"

key_del "$KEY"
finish bounding_box
