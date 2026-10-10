#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:dense_vector:$$:$RANDOM"

start_server

cli SET "{\"key\":\"$KEY\",\"type\":\"dense_vector\",\"value\":[1.0,0.0,0.5]}" >/dev/null
key_lifecycle "$KEY" dense_vector

out="$(extract "$(op "$KEY" DISTANCE '{"vector":[1.0,0.0,0.5],"metric":"cosine"}')")"
python3 -c "import sys; assert abs(float('$out')) < 1e-9, f'expected near 0 got {sys.argv[1]}'" "$out"

_PASSES=$((_PASSES + 1))

key_del "$KEY"
finish dense_vector
