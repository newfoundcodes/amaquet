#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/scenario.sh"

scenario_init

k="$(scenario_key intmax)"
setv "$k" integer 9223372036854775806
assert_eq "$(op "$k" ADD '{"delta":1}' | json_extract x)" 9223372036854775807

k2="$(scenario_key intmin)"
setv "$k2" integer -9223372036854775807
assert_eq "$(op "$k2" ADD '{"delta":-1}' | json_extract x)" -9223372036854775808

u="$(scenario_key uint)"
setv "$u" unsigned_integer 18446744073709551614
assert_eq "$(op "$u" ADD '{"delta":1}' | json_extract x)" 18446744073709551615

b="$(scenario_key bigint)"
setv "$b" big_integer '"99999999999999999999999999999999999999999999999999"'
assert_contains "$(op "$b" ADD '{"value":"1"}')" 100000000000000000000000000000000000000000000000000

d="$(scenario_key dec)"
setv "$d" big_decimal '"0.0000000000000000000000000001"'
assert_contains "$(op "$d" ADD '{"value":"99999999999999999999.9999999999"}')" 99999999999999999999

f="$(scenario_key float)"
setv "$f" float 1e-200
out="$(op "$f" ADD '{"delta":1e-200}' | json_extract x)"

python3 - "$out" <<'PY'
import math,sys
x=float(sys.argv[1]); assert math.isfinite(x) and x>0
PY

server_healthy
echo "PASS numeric edges"
