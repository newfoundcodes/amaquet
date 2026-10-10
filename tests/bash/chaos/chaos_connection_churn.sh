#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/common.sh"

: "${AMAQUET_CHURN_CONNECTIONS:=200}"
: "${AMAQUET_CHURN_PARALLEL:=8}"

start_test_server
export AMAQUET_CLI AMAQUET_URI

seq "$AMAQUET_CHURN_CONNECTIONS" | xargs -P "$AMAQUET_CHURN_PARALLEL" -n1 sh -c '"$AMAQUET_CLI" -uri "$AMAQUET_URI" -timeout 5s PING "{}" >/dev/null' _
amaquet PING '{}' >/dev/null

log "CONNECTION CHURN PASS connections=$AMAQUET_CHURN_CONNECTIONS parallel=$AMAQUET_CHURN_PARALLEL"
