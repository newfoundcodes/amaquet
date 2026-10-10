#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:geo_spatial_index:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"geo_spatial_index\",\"options\":{}}" >/dev/null
key_lifecycle "$KEY" geo_spatial_index

cli OP "{\"key\":\"$KEY\",\"operation\":\"ADD\",\"args\":{\"id\":\"manila\",\"point\":{\"Lat\":14.5995,\"Lon\":120.9842},\"metadata\":{\"country\":\"PH\"}}}" >/dev/null
cli OP "{\"key\":\"$KEY\",\"operation\":\"ADD\",\"args\":{\"id\":\"cebu\",\"point\":{\"Lat\":10.3157,\"Lon\":123.8854},\"metadata\":{\"country\":\"PH\"}}}" >/dev/null

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"RADIUS\",\"args\":{\"center\":{\"Lat\":14.5995,\"Lon\":120.9842},\"radius_meters\":1000,\"limit\":10}}")"
assert_contains "$out" manila "RADIUS"

key_del "$KEY"
finish geo_spatial_index
