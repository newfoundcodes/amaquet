#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:full_text_document:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"full_text_document\",\"options\":{\"id\":\"doc-1\",\"metadata\":{\"source\":\"cli-test\"}}}" >/dev/null
key_lifecycle "$KEY" full_text_document

cli OP "{\"key\":\"$KEY\",\"operation\":\"ADD_TEXT\",\"args\":{\"name\":\"body\",\"value\":\"hello amaquet world\",\"weight\":2}}" >/dev/null
cli OP "{\"key\":\"$KEY\",\"operation\":\"ADD_TAG\",\"args\":{\"name\":\"tags\",\"values\":[\"go\",\"memory\"]}}" >/dev/null
cli OP "{\"key\":\"$KEY\",\"operation\":\"ADD_NUMERIC\",\"args\":{\"name\":\"score\",\"value\":9.5}}" >/dev/null

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"SNAPSHOT\",\"args\":{}}")"
assert_contains "$out" amaquet "SNAPSHOT text"
assert_contains "$out" go "SNAPSHOT tags"

key_del "$KEY"
finish full_text_document
