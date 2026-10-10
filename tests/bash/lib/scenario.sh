#!/usr/bin/env bash
set -Eeuo pipefail

SCENARIO_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCENARIO_LIB_DIR/common.sh"

: "${AMAQUET_SCENARIO_ITERATIONS:=20}"
: "${AMAQUET_FUZZ_SEED:=1337}"

[[ "$AMAQUET_SCENARIO_ITERATIONS" =~ ^[1-9][0-9]*$ ]] || die "AMAQUET_SCENARIO_ITERATIONS must be a positive integer"
[[ "$AMAQUET_FUZZ_SEED" =~ ^[0-9]+$ ]] || die "AMAQUET_FUZZ_SEED must be an integer"

RANDOM=$((AMAQUET_FUZZ_SEED % 32768))

scenario_init() { start_test_server; }

scenario_key() { printf 'scenario:%s:%s:%s' "${1:-key}" "$$" "$RANDOM"; }

server_healthy() {
    local p
    p="$(amaquet PING '{}' | json_extract 'x')"
    assert_eq "$p" PONG "server health"
}

json_len() { python3 -c 'import json,sys; print(len(json.load(sys.stdin)))'; }

json_field() {
    local field="$1"
    python3 -c 'import json,sys; x=json.load(sys.stdin); print(x[sys.argv[1]])' "$field"
}

json_nested() {
    local a="$1" b="$2"
    python3 -c 'import json,sys; x=json.load(sys.stdin); print(x[sys.argv[1]][sys.argv[2]])' "$a" "$b"
}

wait_until_missing() {
    local key="$1" tries="${2:-100}"
    local i

    for ((i = 0; i < tries; i++)); do
        if ! amaquet EXISTS "{\"key\":\"$key\"}" 2>/dev/null | grep -q true; then return 0; fi
        sleep 0.01
    done

    return 1
}

repeat_char() {
    local ch="$1" n="$2"
    python3 - "$ch" "$n" <<'PY'
import sys
print(sys.argv[1]*int(sys.argv[2]), end='')
PY
}

random_b64() {
    local n="${1:-256}"
    python3 - "$n" "$AMAQUET_FUZZ_SEED" <<'PY'
import base64,random,sys
n=int(sys.argv[1]); r=random.Random(int(sys.argv[2])+n)
print(base64.b64encode(bytes(r.randrange(256) for _ in range(n))).decode())
PY
}
