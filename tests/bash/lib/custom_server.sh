#!/usr/bin/env bash
set -Eeuo pipefail

CUSTOM_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

source "$CUSTOM_LIB_DIR/common.sh"
CUSTOM_ADMIN_PORT=""

custom_start_server() {
    local config="$1" proto_port="$2" admin_port="$3"
    ensure_bins

    AMAQUET_TEST_TMP="$(dirname "$config")"
    "$AMAQUET_BIN" -config "$config" >"$AMAQUET_TEST_TMP/server.log" 2>&1 &

    AMAQUET_SERVER_PID=$!
    AMAQUET_OWN_SERVER=1
    CUSTOM_ADMIN_PORT="$admin_port"
    AMAQUET_URI="amaquet://127.0.0.1:$proto_port"

    export AMAQUET_URI
    wait_tcp 127.0.0.1 "$proto_port" 160 || {
        cat "$AMAQUET_TEST_TMP/server.log" >&2 || true
        die "custom server did not start"
    }

    wait_tcp 127.0.0.1 "$admin_port" 160 || {
        cat "$AMAQUET_TEST_TMP/server.log" >&2 || true
        die "custom admin server did not start"
    }
}

custom_stop_server() {
    local old_keep="${AMAQUET_KEEP_TMP:-0}"

    AMAQUET_KEEP_TMP=1 stop_test_server
    AMAQUET_KEEP_TMP="$old_keep"
    AMAQUET_OWN_SERVER=0
    AMAQUET_SERVER_PID=""
}

admin_curl() {
    local token="$1" method="$2" path="$3" body="${4:-}"
    local args=(-sS -H "Authorization: Bearer $token" -H 'Content-Type: application/json' -X "$method")

    [[ -n "$body" ]] && args+=(-d "$body")
    curl "${args[@]}" "http://127.0.0.1:$CUSTOM_ADMIN_PORT$path"
}

admin_status() {
    local token="$1" method="$2" path="$3" body="${4:-}"
    local args=(-sS -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $token" -H 'Content-Type: application/json' -X "$method")

    [[ -n "$body" ]] && args+=(-d "$body")
    curl "${args[@]}" "http://127.0.0.1:$CUSTOM_ADMIN_PORT$path"
}
