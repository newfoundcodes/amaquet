#!/usr/bin/env bash
set -Eeuo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

CLI="${AMAQUET_CLI:-$REPO_ROOT/bin/amaquet-cli}"
SRV="${AMAQUET_BIN:-$REPO_ROOT/bin/amaquet}"
URI="${AMAQUET_URI:-}"

_TMP=""
_PID=""
_PASSES=0

log() {
    printf '[cli-test] %s\n' "$*" >&2;
}

fail() {
    printf '[cli-test] FAIL: %s\n' "$*" >&2
    exit 1
}

need() {
    command -v "$1" >/dev/null 2>&1 || fail "required tool not found: $1";
}

_build_bins() {
    need go
    mkdir -p "$REPO_ROOT/bin"

    (cd "$REPO_ROOT" &&
        go build -o "$SRV" ./cmd/amaquet &&
        go build -o "$CLI" ./cmd/amaquet-cli)
}

_ensure_bins() {
    if [[ ! -x "$SRV" || ! -x "$CLI" ]] ||
        ! "$SRV" -version >/dev/null 2>&1 ||
        ! "$CLI" -h >/dev/null 2>&1; then
        _build_bins
    fi

    need python3
}

_free_port() {
    python3 - <<'PY'
import socket
s=socket.socket(); s.bind(('127.0.0.1',0)); print(s.getsockname()[1]); s.close()
PY
}

_wait_tcp() {
    python3 - "$1" "$2" "${3:-100}" <<'PY'
import socket,sys,time
h,p,n=sys.argv[1],int(sys.argv[2]),int(sys.argv[3])
for _ in range(n):
    try:
        with socket.create_connection((h,p),0.15): sys.exit(0)
    except OSError: time.sleep(.05)
sys.exit(1)
PY
}

start_server() {
    _ensure_bins
    [[ -n "$URI" ]] && return 0

    _TMP="$(mktemp -d -t amaquet-cli-test.XXXXXX)"
    local port adm

    port="$(_free_port)"
    adm="$(_free_port)"

    cat >"$_TMP/cfg.json" <<JSON
{"protocol":{"host":"127.0.0.1","port":$port,"tls":false,"require_auth":false},
 "admin":{"host":"127.0.0.1","port":$adm},
 "security":{"bootstrap_token":"cli-test"},
 "persistence":{"aof_enabled":false,"aof_path":"$_TMP/aof","fsync":"no"},
 "limits":{"max_connections":512,"max_payload_bytes":8388608},
 "data_dir":"$_TMP/data"}
JSON

    "$SRV" -config "$_TMP/cfg.json" >"$_TMP/server.log" 2>&1 &

    _PID=$!
    URI="amaquet://127.0.0.1:$port"

    export URI CLI SRV
    _wait_tcp 127.0.0.1 "$port" || {
        cat "$_TMP/server.log" >&2 || true
        fail "server did not start"
    }

    "$CLI" -uri "$URI" -timeout 5s PING '{}' >/dev/null
}

_stop_server() {
    if [[ -n "$_PID" ]]; then
        kill "$_PID" 2>/dev/null || true

        local i
        for i in {1..10}; do
            kill -0 "$_PID" 2>/dev/null || break
            sleep 0.05
        done

        kill -0 "$_PID" 2>/dev/null && kill -KILL "$_PID" 2>/dev/null || true
        wait "$_PID" 2>/dev/null || true

        _PID=""
    fi

    [[ -n "$_TMP" && "${AMAQUET_KEEP_TMP:-0}" != 1 ]] && rm -rf "$_TMP" || true
}

trap _stop_server EXIT INT TERM

cli() {
    local timeout="${AMAQUET_TIMEOUT:-5s}"

    printf "%q " "amaquet-cli" -uri "$URI" -timeout "$timeout" "$@" >&2
    printf '\n' >&2

    "$CLI" -uri "$URI" -timeout "$timeout" "$@"
}

op() {
    local key="$1" operation="$2" payload="${3:-{}}"
    cli OP "{\"key\":\"$key\",\"operation\":\"$operation\",\"args\":$payload}"
}

extract() {
    local val="${1:-}"
    if [[ $# -eq 0 ]]; then
        val="$(cat)"
    fi

    python3 -c '
import json, sys

raw = sys.stdin.read().strip()
if not raw:
    sys.exit(0)
try:
    d = json.loads(raw)
except Exception:
    print(raw)
    sys.exit(0)

field = sys.argv[1] if len(sys.argv) > 1 and sys.argv[1] else None
if field and isinstance(d, dict) and field in d:
    d = d[field]

if isinstance(d, bool):
    print("true" if d else "false")
elif isinstance(d, (int, float, str)):
    print(d)
elif d is None:
    print("null")
else:
    print(json.dumps(d))
' "${2:-}" <<<"$val"
}

assert_eq() {
    [[ "$1" == "$2" ]] || fail "assert_eq: expected [$2] got [$1]${3:+ ($3)}"
    _PASSES=$((_PASSES + 1))
}

assert_contains() {
    [[ "$1" == *"$2"* ]] || fail "assert_contains: [$2] not in [$1]"
    _PASSES=$((_PASSES + 1))
}

assert_true() {
    [[ "$1" == true || "$1" == True || "$1" == 1 ]] || fail "assert_true: got [$1]"
    _PASSES=$((_PASSES + 1))
}

assert_false() {
    [[ "$1" == false || "$1" == False || "$1" == 0 ]] || fail "assert_false: got [$1]"
    _PASSES=$((_PASSES + 1))
}

assert_nonempty() {
    [[ -n "$1" && "$1" != null && "$1" != None ]] || fail "assert_nonempty: empty value"
    _PASSES=$((_PASSES + 1))
}

assert_ge() {
    python3 -c "import sys; sys.exit(0 if float('$1') >= float('$2') else 1)" ||
        fail "assert_ge: $1 < $2"
    _PASSES=$((_PASSES + 1))
}

assert_command_fails() {
    if "$@" >/dev/null 2>&1; then fail "command unexpectedly succeeded: $*"; fi
    _PASSES=$((_PASSES + 1))
}

key_lifecycle() {
    local key="$1" type="$2" out ttl

    out="$(extract "$(cli TYPE "{\"key\":\"$key\"}")")"
    assert_eq "$out" "$type" "TYPE"

    out="$(extract "$(cli EXISTS "{\"key\":\"$key\"}")")"
    assert_true "$out"

    cli EXPIRE "{\"key\":\"$key\",\"ttl_ms\":5000}" >/dev/null
    ttl="$(extract "$(cli TTL "{\"key\":\"$key\"}")")"

    [[ "$ttl" -gt 0 && "$ttl" -le 5000 ]] || fail "invalid TTL: $ttl"
    _PASSES=$((_PASSES + 1))

    cli PERSIST "{\"key\":\"$key\"}" >/dev/null
    ttl="$(extract "$(cli TTL "{\"key\":\"$key\"}")")"

    assert_eq "$ttl" -1 "PERSIST"
}

key_del() {
    local key="$1" n
    n="$(extract "$(cli DEL "{\"keys\":[\"$key\"]}")")"

    assert_eq "$n" 1 "DEL"
}

finish() {
    log "PASS ${1:-} ($_PASSES assertions)";
}
