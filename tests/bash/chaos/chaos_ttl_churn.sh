#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/common.sh"
: "${AMAQUET_TTL_CHURN:=2000}"

start_test_server
setv chaos:ttl integer 1

for ((i = 0; i < AMAQUET_TTL_CHURN; i++)); do
    amaquet EXPIRE '{"key":"chaos:ttl","ttl_ms":60000}' >/dev/null
    amaquet PERSIST '{"key":"chaos:ttl"}' >/dev/null
done

assert_true "$(amaquet EXISTS '{"key":"chaos:ttl"}' | json_extract x)"
amaquet PING '{}' >/dev/null

log "TTL CHURN PASS cycles=$AMAQUET_TTL_CHURN"
