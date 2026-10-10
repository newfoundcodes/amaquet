#!/usr/bin/env bash
set -Eeuo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
: "${AMAQUET_GO_FUZZ_TIME:=5s}"
cd "$ROOT"

targets=()
while IFS= read -r target; do
  targets+=("$target")

done < <(grep -RhoE '^func (Fuzz[A-Za-z0-9_]+)\(' internal --include='*_fuzz_test.go' | sed -E 's/^func (Fuzz[A-Za-z0-9_]+)\(.*/\1/' | sort -u)
(( ${#targets[@]} > 0 )) || { echo "no native Go fuzz targets found" >&2; exit 1; }

for target in "${targets[@]}"; do
  pkg="$(grep -Rl "func $target(" internal --include='*_fuzz_test.go' | head -1 | xargs dirname)"

  printf '\n==> %s (%s, %s)\n' "$target" "$pkg" "$AMAQUET_GO_FUZZ_TIME"
  go test "./$pkg" -run='^$' -fuzz="^${target}$" -fuzztime="$AMAQUET_GO_FUZZ_TIME"
done
