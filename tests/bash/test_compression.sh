#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$HERE/lib/common.sh"

# Compression changed the server binary, so this feature test always rebuilds.
build_bins
start_test_server

large_json="$(
    python3 - <<'PY'
import json
print(json.dumps("Amaquet-compression-" * 2500))
PY
)"
setv "compression:large" "utf8_string" "$large_json"

mem="$(amaquet MEMORY '{"key":"compression:large"}')"
python3 - "$mem" <<'PY'
import json,sys
x=json.loads(sys.argv[1])
assert x["compressed"] is True, x
assert x["stored_bytes"] < x["original_bytes"], x
assert x["savings_percent"] > 50, x
assert x["algorithm"] in {"lz4","rle","deflate-fast"}, x
PY

small='"short"'
setv "compression:small" "utf8_string" "$small"
small_mem="$(amaquet MEMORY '{"key":"compression:small"}')"

python3 - "$small_mem" <<'PY'
import json,sys
x=json.loads(sys.argv[1])
assert x["compressed"] is False, x
PY

value="$(amaquet GET '{"key":"compression:large"}')"
assert_contains "$value" "Amaquet-compression-"

info="$(amaquet INFO '{}')"
python3 - "$info" <<'PY'
import json,sys
x=json.loads(sys.argv[1])
c=x["compression"]
assert c["compressed_keys"] >= 1, x
assert c["savings_bytes"] > 0, x
PY

log "PASS adaptive compression"
