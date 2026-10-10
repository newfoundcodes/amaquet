#!/usr/bin/env bash
set -Eeuo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

: "${AMAQUET_BENCH_TIME:=1s}"
out="$(go test ./internal/core -run '^$' -bench 'Benchmark(SetGet|ConcurrentSetGet|ExpirePersistChurn|ScanPage)$' -benchmem -benchtime "$AMAQUET_BENCH_TIME")"

printf '%s\n' "$out"

# Optional release gate for the single-threaded SET+GET path. The threshold is
# intentionally opt-in because runner hardware differs substantially.
if [[ -n "${AMAQUET_BENCH_SETGET_MAX_NS:-}" ]]; then
  actual="$(printf '%s\n' "$out" | awk '/^BenchmarkSetGet-/ {print $(NF-3); exit}')"
  [[ -n "$actual" ]] || { echo 'could not parse BenchmarkSetGet' >&2; exit 1; }
  python3 - "$actual" "$AMAQUET_BENCH_SETGET_MAX_NS" <<'PY'
import sys
actual=float(sys.argv[1]); limit=float(sys.argv[2])
if actual > limit:
    raise SystemExit(f"BenchmarkSetGet regression: {actual} ns/op > {limit} ns/op")
PY
fi
