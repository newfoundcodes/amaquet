#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

"$HERE/protocol/test_uri_and_commands.sh"
"$HERE/protocol/fuzz_frames.sh"
