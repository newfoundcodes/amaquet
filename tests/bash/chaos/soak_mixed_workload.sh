#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/common.sh"

: "${AMAQUET_SOAK_SECONDS:=10}"
: "${AMAQUET_SOAK_KEYS:=128}"

start_test_server
end=$((SECONDS + AMAQUET_SOAK_SECONDS))
i=0

while ((SECONDS < end)); do
    k="soak:$((i % AMAQUET_SOAK_KEYS))"

    case $((i % 6)) in
        0) amaquet SET "{\"key\":\"$k\",\"type\":\"integer\",\"value\":$i}" >/dev/null ;;
        1) amaquet GET "{\"key\":\"$k\"}" >/dev/null 2>&1 || true ;;
        2) amaquet SET "{\"key\":\"$k:text\",\"type\":\"utf8_string\",\"value\":\"value-$i-$(printf '%064d' 0)\"}" >/dev/null ;;
        3) amaquet EXPIRE "{\"key\":\"$k:text\",\"ttl_ms\":250}" >/dev/null 2>&1 || true ;;
        4) amaquet SCAN '{"cursor":"","count":32,"prefix":"soak:"}' >/dev/null ;;
        5) amaquet DEL "{\"keys\":[\"$k\"]}" >/dev/null ;;
    esac

    i=$((i + 1))
done

amaquet PING '{}' >/dev/null
log "SOAK PASS operations=$i seconds=$AMAQUET_SOAK_SECONDS"
