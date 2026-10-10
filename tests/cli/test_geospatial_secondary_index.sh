#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:geospatial_secondary_index:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"geospatial_secondary_index\",\"options\":{}}" >/dev/null
key_lifecycle "$KEY" geospatial_secondary_index

cli OP "{\"key\":\"$KEY\",\"operation\":\"ADD\",\"args\":{\"id\":\"manila\",\"point\":{\"Lat\":14.5995,\"Lon\":120.9842},\"metadata\":{\"country\":\"PH\"}}}" >/dev/null

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"RADIUS\",\"args\":{\"center\":{\"Lat\":14.5995,\"Lon\":120.9842},\"radius_meters\":100,\"limit\":5}}")"
assert_contains "$out" manila "RADIUS"

key_del "$KEY"
finish geospatial_secondary_index
