#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:quantized_vector:$$:$RANDOM"

start_server

cli SET "{\"key\":\"$KEY\",\"type\":\"quantized_vector\",\"value\":{\"Data\":[1,2,-3],\"Scale\":0.1,\"ZeroPoint\":0}}" >/dev/null
key_lifecycle "$KEY" quantized_vector

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"DEQUANTIZE\",\"args\":{}}")"
assert_contains "$out" 0.1 "DEQUANTIZE scale"

key_del "$KEY"
finish quantized_vector
