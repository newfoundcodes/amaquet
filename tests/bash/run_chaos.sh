#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

for test in "$HERE"/chaos/*.sh; do
    printf '\n==> %s\n' "$(basename "$test")"
    "$test"
done
