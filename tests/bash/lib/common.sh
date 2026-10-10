#!/usr/bin/env bash
set -Eeuo pipefail

AMAQUET_TEST_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
AMAQUET_BIN="${AMAQUET_BIN:-$AMAQUET_TEST_ROOT/bin/amaquet}"
AMAQUET_CLI="${AMAQUET_CLI:-$AMAQUET_TEST_ROOT/bin/amaquet-cli}"
AMAQUET_URI="${AMAQUET_URI:-}"
AMAQUET_TEST_TMP="${AMAQUET_TEST_TMP:-}"
AMAQUET_OWN_SERVER=0
AMAQUET_SERVER_PID=""
AMAQUET_TEST_PASSES=0

log() {
    printf '[amaquet-test] %s\n' "$*" >&2
}

die() {
    printf '[amaquet-test] FAIL: %s\n' "$*" >&2
    exit 1
}

need() {
    command -v "$1" >/dev/null 2>&1 || die "required command not found: $1"
}

build_bins() {
    need go
    mkdir -p "$AMAQUET_TEST_ROOT/bin"
    (cd "$AMAQUET_TEST_ROOT" && go build -o "$AMAQUET_BIN" ./cmd/amaquet && go build -o "$AMAQUET_CLI" ./cmd/amaquet-cli)
}

ensure_bins() {
    if [[ ! -x "$AMAQUET_BIN" || ! -x "$AMAQUET_CLI" ]] ||
        ! "$AMAQUET_BIN" -version >/dev/null 2>&1 ||
        ! "$AMAQUET_CLI" -h >/dev/null 2>&1; then
        build_bins
    fi
    need python3
}

free_port() {
    python3 - <<'PYPORT'
import socket
s=socket.socket(); s.bind(('127.0.0.1',0)); print(s.getsockname()[1]); s.close()
PYPORT
}

wait_tcp() {
    python3 - "$1" "$2" "${3:-100}" <<'PYWAIT'
import socket,sys,time
h,p,n=sys.argv[1],int(sys.argv[2]),int(sys.argv[3])
for _ in range(n):
    try:
        with socket.create_connection((h,p),0.15): sys.exit(0)
    except OSError: time.sleep(.05)
sys.exit(1)
PYWAIT
}

start_test_server() {
    ensure_bins

    [[ -n "$AMAQUET_URI" ]] && return 0
    AMAQUET_TEST_TMP="${AMAQUET_TEST_TMP:-$(mktemp -d -t amaquet-bash.XXXXXX)}"

    local port admin_port
    port="$(free_port)"
    admin_port="$(free_port)"

    cat >"$AMAQUET_TEST_TMP/amaquet.json" <<JSONCFG
{"protocol":{"host":"127.0.0.1","port":$port,"tls":false,"require_auth":false},"admin":{"host":"127.0.0.1","port":$admin_port},"security":{"bootstrap_token":"bash-test-bootstrap"},"persistence":{"aof_enabled":false,"aof_path":"$AMAQUET_TEST_TMP/amaquet.aof","fsync":"no"},"limits":{"max_connections":1024,"max_payload_bytes":8388608},"data_dir":"$AMAQUET_TEST_TMP/data"}
JSONCFG

    "$AMAQUET_BIN" -config "$AMAQUET_TEST_TMP/amaquet.json" >"$AMAQUET_TEST_TMP/server.log" 2>&1 &

    AMAQUET_SERVER_PID=$!
    AMAQUET_OWN_SERVER=1
    AMAQUET_URI="amaquet://127.0.0.1:$port"

    export AMAQUET_URI AMAQUET_BIN AMAQUET_CLI AMAQUET_TEST_TMP
    wait_tcp 127.0.0.1 "$port" || {
        cat "$AMAQUET_TEST_TMP/server.log" >&2 || true
        die "server did not start"
    }

    amaquet PING '{}' >/dev/null
}

stop_test_server() {
    if [[ "$AMAQUET_OWN_SERVER" == 1 && -n "$AMAQUET_SERVER_PID" ]]; then
        kill "$AMAQUET_SERVER_PID" 2>/dev/null || true

        # The production server drains active connections on SIGTERM. Test instances are
        # disposable, so cap teardown time to keep the per-type isolated suite practical.
        local i
        for i in {1..10}; do
            kill -0 "$AMAQUET_SERVER_PID" 2>/dev/null || break
            sleep 0.05
        done

        if kill -0 "$AMAQUET_SERVER_PID" 2>/dev/null; then
            kill -KILL "$AMAQUET_SERVER_PID" 2>/dev/null || true
        fi
        wait "$AMAQUET_SERVER_PID" 2>/dev/null || true
    fi

    if [[ "$AMAQUET_OWN_SERVER" == 1 && -n "$AMAQUET_TEST_TMP" && "${AMAQUET_KEEP_TMP:-0}" != 1 ]]; then
        rm -rf "$AMAQUET_TEST_TMP"
    fi
}

trap stop_test_server EXIT INT TERM

amaquet() {
    "$AMAQUET_CLI" -uri "$AMAQUET_URI" -timeout "${AMAQUET_TIMEOUT:-5s}" "$@"
}

json_extract() {
    local expr="$1"

    if command -v jq >/dev/null 2>&1 && [[ "$expr" == "x" ]]; then
        jq -r '.'
    else
        python3 -c 'import json,sys; x=json.load(sys.stdin); print(eval(sys.argv[1], {"__builtins__":{}}, {"x":x}))' "$expr"
    fi
}

assert_eq() {
    [[ "$1" == "$2" ]] || die "expected [$2], got [$1]${3:+ ($3)}"
    AMAQUET_TEST_PASSES=$((AMAQUET_TEST_PASSES + 1))
}

assert_contains() {
    [[ $# -eq 2 ]] || die "assert_contains expects 2 arguments, got $#"
    [[ "$1" == *"$2"* ]] || die "expected output to contain [$2], got [$1]"
    AMAQUET_TEST_PASSES=$((AMAQUET_TEST_PASSES + 1))
}

assert_true() {
    [[ "$1" == true || "$1" == True || "$1" == 1 ]] || die "expected true, got [$1]"
    AMAQUET_TEST_PASSES=$((AMAQUET_TEST_PASSES + 1))
}

assert_false() {
    [[ "$1" == false || "$1" == False || "$1" == 0 ]] || die "expected false, got [$1]"
    AMAQUET_TEST_PASSES=$((AMAQUET_TEST_PASSES + 1))
}

assert_nonempty() {
    [[ -n "$1" && "$1" != null && "$1" != None ]] || die "expected non-empty value"
    AMAQUET_TEST_PASSES=$((AMAQUET_TEST_PASSES + 1))
}

assert_command_fails() {
    if "$@" >/dev/null 2>&1; then die "command unexpectedly succeeded: $*"; fi
    AMAQUET_TEST_PASSES=$((AMAQUET_TEST_PASSES + 1))
}

op() {
    local key="$1" operation="$2" args="{}"
    [[ $# -ge 3 ]] && args="$3"
    amaquet OP "{\"key\":\"$key\",\"operation\":\"$operation\",\"args\":$args}"
}

create() {
    local key="$1" type="$2" options="{}"
    [[ $# -ge 3 ]] && options="$3"
    amaquet CREATE "{\"key\":\"$key\",\"type\":\"$type\",\"options\":$options}" >/dev/null
}

setv() {
    local key="$1" type="$2" value="$3"
    amaquet SET "{\"key\":\"$key\",\"type\":\"$type\",\"value\":$value}" >/dev/null
}

setup_type() {
    local key="$1" type="$2"
    case "$type" in
        binary_string) setv "$key" "$type" '"AAECaGVsbG8="' ;;
        utf8_string) setv "$key" "$type" '"Hello, Amaquet 👋"' ;;
        integer) setv "$key" "$type" '42' ;;
        unsigned_integer) setv "$key" "$type" '42' ;;
        float) setv "$key" "$type" '3.14159' ;;
        decimal) setv "$key" "$type" '"12345.6789"' ;;
        boolean) setv "$key" "$type" 'true' ;;
        null) setv "$key" "$type" 'null' ;;
        timestamp) setv "$key" "$type" '"2026-10-01T00:00:00Z"' ;;
        duration) setv "$key" "$type" '"5s"' ;;
        uuid) setv "$key" "$type" '"550e8400-e29b-41d4-a716-446655440000"' ;;
        big_integer) setv "$key" "$type" '"1234567890123456789012345678901234567890"' ;;
        big_decimal) setv "$key" "$type" '"12345678901234567890.000000000000000001"' ;;
        symbol) setv "$key" "$type" '"ACTIVE"' ;;
        geo_point) setv "$key" "$type" '{"Lat":14.5995,"Lon":120.9842}' ;;
        bounding_box) setv "$key" "$type" '{"MinLat":14.0,"MinLon":120.0,"MaxLat":15.0,"MaxLon":122.0}' ;;
        polygon) setv "$key" "$type" '[{"Lat":14.5,"Lon":120.9},{"Lat":14.7,"Lon":120.9},{"Lat":14.6,"Lon":121.1}]' ;;
        dense_vector) setv "$key" "$type" '[1.0,0.0,0.5]' ;;
        sparse_vector) setv "$key" "$type" '{"dim":8,"values":{"1":0.5,"5":1.25}}' ;;
        quantized_vector) setv "$key" "$type" '{"Data":[1,2,-3],"Scale":0.1,"ZeroPoint":0}' ;;
        cidr_network) setv "$key" "$type" '"10.0.0.0/8"' ;;
        url) setv "$key" "$type" '"https://example.com:8443/path?q=amaquet"' ;;
        list) create "$key" "$type" '{}' ;;
        array) create "$key" "$type" '{"typed":true,"element_type":"utf8_string"}' ;;
        deque) create "$key" "$type" '{"capacity":8}' ;;
        ring_buffer) create "$key" "$type" '{"capacity":4}' ;;
        tuple) create "$key" "$type" '{"values":[{"type":"utf8_string","value":"alpha"},{"type":"integer","value":7}]}' ;;
        set) create "$key" "$type" '{}' ;;
        sorted_set) create "$key" "$type" '{}' ;;
        ordered_set) create "$key" "$type" '{}' ;;
        multiset) create "$key" "$type" '{}' ;;
        hashmap) create "$key" "$type" '{}' ;;
        ordered_map) create "$key" "$type" '{}' ;;
        multimap) create "$key" "$type" '{}' ;;
        json) create "$key" "$type" '{}' ;;
        messagepack) create "$key" "$type" '{}' ;;
        cbor) create "$key" "$type" '{}' ;;
        record) create "$key" "$type" '{"schema":[{"name":"name","type":"utf8_string","nullable":false},{"name":"age","type":"integer","nullable":true}]}' ;;
        matrix) create "$key" "$type" '{"rows":2,"cols":2,"dtype":"float64","data":[1,2,3,4]}' ;;
        tensor) create "$key" "$type" '{"shape":[2,2,2],"dtype":"float64","data":[0,1,2,3,4,5,6,7]}' ;;
        fifo_queue) create "$key" "$type" '{}' ;;
        lifo_stack) create "$key" "$type" '{}' ;;
        priority_queue) create "$key" "$type" '{}' ;;
        blocking_queue) create "$key" "$type" '{}' ;;
        delayed_queue) create "$key" "$type" '{}' ;;
        reliable_queue) create "$key" "$type" '{"max_attempts":3}' ;;
        stream) create "$key" "$type" '{}' ;;
        consumer_group) create "$key" "$type" '{"name":"workers"}' ;;
        pubsub) create "$key" "$type" '{}' ;;
        persistent_topic) create "$key" "$type" '{"max_messages":100}' ;;
        event_log) create "$key" "$type" '{}' ;;
        bitmap) create "$key" "$type" '{}' ;;
        bit_field) create "$key" "$type" '{"size_bytes":8}' ;;
        bit_set) create "$key" "$type" '{}' ;;
        roaring_bitmap) create "$key" "$type" '{}' ;;
        hyperloglog) create "$key" "$type" '{"precision":12}' ;;
        bloom_filter) create "$key" "$type" '{"expected":1000,"false_positive_rate":0.01}' ;;
        counting_bloom_filter) create "$key" "$type" '{"expected":1000,"false_positive_rate":0.01}' ;;
        cuckoo_filter) create "$key" "$type" '{"capacity":1024}' ;;
        count_min_sketch) create "$key" "$type" '{"width":256,"depth":4}' ;;
        top_k) create "$key" "$type" '{"k":5}' ;;
        t_digest) create "$key" "$type" '{"compression":100}' ;;
        timeseries) create "$key" "$type" '{"labels":{"service":"api","env":"test"},"retention_ms":3600000}' ;;
        counter_series) create "$key" "$type" '{"labels":{"metric":"requests"}}' ;;
        gauge_series) create "$key" "$type" '{"labels":{"metric":"cpu"}}' ;;
        histogram) create "$key" "$type" '{"bounds":[1,5,10,50,100]}' ;;
        timeseries_labels) create "$key" "$type" '{"labels":{"env":"test","region":"local"}}' ;;
        aggregated_series) create "$key" "$type" '{"bucket_ms":1000,"aggregation":"avg"}' ;;
        geo_spatial_index) create "$key" "$type" '{}' ;;
        vector_set) create "$key" "$type" '{"dim":3,"metric":"cosine"}' ;;
        vector_metadata) create "$key" "$type" '{}' ;;
        btree_index) create "$key" "$type" '{"degree":8}' ;;
        hash_index) create "$key" "$type" '{}' ;;
        radix_tree) create "$key" "$type" '{}' ;;
        trie) create "$key" "$type" '{}' ;;
        inverted_index) create "$key" "$type" '{}' ;;
        hnsw_index) create "$key" "$type" '{"dim":3,"metric":"cosine","m":8,"ef_search":32}' ;;
        flat_vector_index) create "$key" "$type" '{"dim":3,"metric":"cosine"}' ;;
        geospatial_secondary_index) create "$key" "$type" '{}' ;;
        secondary_index) create "$key" "$type" '{}' ;;
        full_text_document) create "$key" "$type" '{"id":"doc-1","metadata":{"source":"bash-test"}}' ;;
        tag_field) create "$key" "$type" '{"name":"tags","values":["go","database"]}' ;;
        numeric_field) create "$key" "$type" '{"name":"score","value":42.5}' ;;
        text_field) create "$key" "$type" '{"name":"body","value":"hello amaquet","weight":1.5}' ;;
        adjacency_set) create "$key" "$type" '{}' ;;
        atomic_counter) create "$key" "$type" '{"initial":10}' ;;
        lease) create "$key" "$type" '{}' ;;
        lock) create "$key" "$type" '{}' ;;
        barrier) create "$key" "$type" '{"target":1}' ;;
        semaphore) create "$key" "$type" '{"limit":2}' ;;
        rate_limit_bucket) create "$key" "$type" '{"capacity":5,"refill_per_second":2}' ;;
        *) die "unknown test type: $type" ;; esac
}

common_key_contract() {
    local key="$1" type="$2" actual exists ttl

    actual="$(amaquet TYPE "{\"key\":\"$key\"}" | json_extract 'x')"
    assert_eq "$actual" "$type" TYPE

    exists="$(amaquet EXISTS "{\"key\":\"$key\"}" | json_extract 'x')"
    assert_true "$exists"

    amaquet EXPIRE "{\"key\":\"$key\",\"ttl_ms\":5000}" >/dev/null
    ttl="$(amaquet TTL "{\"key\":\"$key\"}" | json_extract 'x')"
    [[ "$ttl" -gt 0 && "$ttl" -le 5000 ]] || die "invalid TTL $ttl"

    amaquet PERSIST "{\"key\":\"$key\"}" >/dev/null
    ttl="$(amaquet TTL "{\"key\":\"$key\"}" | json_extract 'x')"
    assert_eq "$ttl" -1 PERSIST
}

run_type_semantics() {
    local key="$1" type="$2" out id
    case "$type" in
        binary_string)
            out="$(amaquet GET "{\"key\":\"$key\"}")"
            assert_contains "$out" "\"type\": \"$type\""
            ;;
        utf8_string)
            out="$(amaquet GET "{\"key\":\"$key\"}")"
            assert_contains "$out" "\"type\": \"$type\""
            ;;
        integer)
            out="$(op "$key" ADD '{"delta":8}' | json_extract 'x')"
            assert_eq "$out" 50
            ;;
        unsigned_integer)
            out="$(op "$key" ADD '{"delta":8}' | json_extract 'x')"
            assert_eq "$out" 50
            ;;
        float)
            out="$(op "$key" ADD '{"delta":0.5}' | json_extract 'x')"
            assert_eq "$out" 3.64159
            ;;
        decimal)
            op "$key" ADD '{"value":"0.3211"}' >/dev/null
            out="$(op "$key" CMP '{"value":"12346.0000"}' | json_extract 'x')"
            assert_eq "$out" 0
            ;;
        boolean)
            out="$(op "$key" TOGGLE '{}' | json_extract 'x')"
            assert_false "$out"
            ;;
        null)
            out="$(amaquet GET "{\"key\":\"$key\"}")"
            assert_contains "$out" "\"type\": \"$type\""
            ;;
        timestamp)
            out="$(amaquet GET "{\"key\":\"$key\"}")"
            assert_contains "$out" "\"type\": \"$type\""
            ;;
        duration)
            out="$(op "$key" ADD '{"delta_ms":1000}' | json_extract 'x')"
            assert_contains "$out" 6s
            ;;
        uuid)
            out="$(amaquet GET "{\"key\":\"$key\"}")"
            assert_contains "$out" "\"type\": \"$type\""
            ;;
        big_integer)
            out="$(op "$key" ADD '{"value":"10"}' | json_extract 'x')"
            assert_contains "$out" 1234567890123456789012345678901234567900
            ;;
        big_decimal)
            op "$key" ADD '{"value":"0.3211"}' >/dev/null
            out="$(op "$key" CMP '{"value":"12345678901234567890.321100000000000001"}' | json_extract 'x')"
            assert_eq "$out" 0
            ;;
        symbol)
            out="$(amaquet GET "{\"key\":\"$key\"}")"
            assert_contains "$out" "\"type\": \"$type\""
            ;;
        list)
            op "$key" PUSH_BACK '{"value":{"type":"utf8_string","value":"alpha"}}' >/dev/null
            op "$key" PUSH_FRONT '{"value":{"type":"integer","value":7}}' >/dev/null

            out="$(op "$key" LEN '{}' | json_extract 'x')"
            assert_eq "$out" 2

            out="$(op "$key" RANGE '{"start":0,"stop":-1}')"
            assert_contains "$out" alpha
            ;;
        array)
            op "$key" APPEND '{"value":{"type":"utf8_string","value":"alpha"}}' >/dev/null
            out="$(op "$key" GET '{"index":0}')"

            assert_contains "$out" alpha
            ;;
        deque)
            op "$key" PUSH_BACK '{"value":{"type":"utf8_string","value":"alpha"}}' >/dev/null
            op "$key" PUSH_FRONT '{"value":{"type":"integer","value":7}}' >/dev/null

            out="$(op "$key" LEN '{}' | json_extract 'x')"
            assert_eq "$out" 2
            ;;
        ring_buffer)
            for v in a b c d e; do op "$key" PUSH "{\"value\":{\"type\":\"utf8_string\",\"value\":\"$v\"}}" >/dev/null; done
            out="$(op "$key" LEN '{}' | json_extract 'x')"

            assert_eq "$out" 4
            ;;
        tuple)
            out="$(op "$key" LEN '{}' | json_extract 'x')"
            assert_eq "$out" 2

            out="$(op "$key" GET '{"index":0}')"
            assert_contains "$out" alpha
            ;;
        set)
            out="$(op "$key" ADD '{"value":{"type":"utf8_string","value":"alpha"}}' | json_extract 'x')"
            assert_true "$out"

            out="$(op "$key" CONTAINS '{"value":{"type":"utf8_string","value":"alpha"}}' | json_extract 'x')"
            assert_true "$out"
            ;;
        sorted_set)
            op "$key" ADD '{"member":{"type":"utf8_string","value":"alpha"},"score":10.5}' >/dev/null
            out="$(op "$key" SCORE '{"member":{"type":"utf8_string","value":"alpha"}}' | json_extract 'x')"

            assert_eq "$out" 10.5
            ;;
        ordered_set)
            op "$key" ADD '{"value":{"type":"utf8_string","value":"alpha"}}' >/dev/null
            out="$(op "$key" VALUES '{}')"

            assert_contains "$out" alpha
            ;;
        multiset)
            op "$key" ADD '{"value":{"type":"utf8_string","value":"alpha"},"count":3}' >/dev/null
            out="$(op "$key" COUNT '{"value":{"type":"utf8_string","value":"alpha"}}' | json_extract 'x')"

            assert_eq "$out" 3
            ;;
        hashmap)
            op "$key" SET '{"field":"name","value":{"type":"utf8_string","value":"alpha"}}' >/dev/null
            out="$(op "$key" GET '{"field":"name"}')"

            assert_contains "$out" alpha
            ;;
        ordered_map)
            op "$key" SET '{"field":"name","value":{"type":"utf8_string","value":"alpha"}}' >/dev/null
            out="$(op "$key" GET '{"field":"name"}')"

            assert_contains "$out" alpha
            ;;
        multimap)
            op "$key" ADD '{"field":"tag","value":{"type":"utf8_string","value":"alpha"}}' >/dev/null
            out="$(op "$key" GET '{"field":"tag"}')"

            assert_contains "$out" alpha
            ;;
        json)
            op "$key" SET '{"value":{"hello":"world","n":1}}' >/dev/null
            out="$(op "$key" GET '{}')"

            assert_contains "$out" world
            ;;
        messagepack)
            out="$(amaquet GET "{\"key\":\"$key\"}")"
            assert_contains "$out" "\"type\": \"$type\""
            ;;
        cbor)
            out="$(amaquet GET "{\"key\":\"$key\"}")"
            assert_contains "$out" "\"type\": \"$type\""
            ;;
        record)
            op "$key" SET '{"field":"name","value":{"type":"utf8_string","value":"alpha"}}' >/dev/null
            out="$(op "$key" GET '{"field":"name"}')"

            assert_contains "$out" alpha
            ;;
        matrix)
            out="$(op "$key" GET '{"indices":[1,1]}' | json_extract 'x')"
            assert_eq "$out" 4

            op "$key" SET '{"indices":[0,1],"value":9}' >/dev/null
            ;;
        tensor)
            out="$(op "$key" GET '{"indices":[1,1,1]}' | json_extract 'x')"
            assert_eq "$out" 7

            op "$key" SET '{"indices":[0,0,0],"value":9}' >/dev/null
            ;;
        fifo_queue)
            op "$key" ENQUEUE '{"value":{"type":"utf8_string","value":"alpha"}}' >/dev/null
            out="$(op "$key" PEEK '{}')"

            assert_contains "$out" alpha
            op "$key" DEQUEUE '{}' >/dev/null
            ;;
        lifo_stack)
            op "$key" PUSH '{"value":{"type":"utf8_string","value":"alpha"}}' >/dev/null
            out="$(op "$key" PEEK '{}')"

            assert_contains "$out" alpha
            op "$key" POP '{}' >/dev/null
            ;;
        priority_queue)
            op "$key" ENQUEUE '{"value":{"type":"utf8_string","value":"alpha"},"priority":1}' >/dev/null
            out="$(op "$key" LEN '{}' | json_extract 'x')"

            assert_eq "$out" 1
            op "$key" DEQUEUE '{}' >/dev/null
            ;;
        blocking_queue)
            op "$key" ENQUEUE '{"value":{"type":"utf8_string","value":"alpha"}}' >/dev/null
            out="$(op "$key" DEQUEUE '{}')"

            assert_contains "$out" alpha
            ;;
        delayed_queue)
            op "$key" ENQUEUE '{"id":"job1","value":{"type":"utf8_string","value":"alpha"},"available_at_ms":0}' >/dev/null
            out="$(op "$key" POP_READY '{}')"

            assert_contains "$out" job1
            ;;
        reliable_queue)
            op "$key" ENQUEUE '{"id":"job1","value":{"type":"utf8_string","value":"alpha"}}' >/dev/null

            out="$(op "$key" CLAIM '{"lease_ms":5000}')"
            assert_contains "$out" job1

            out="$(op "$key" ACK '{"id":"job1"}' | json_extract 'x')"
            assert_true "$out"
            ;;
        stream)
            out="$(op "$key" ADD '{"fields":{"message":{"type":"utf8_string","value":"alpha"}}}')"
            assert_nonempty "$out"

            out="$(op "$key" RANGE '{"count":10}')"
            assert_contains "$out" alpha

            op "$key" GROUP_CREATE '{"group":"g1"}' >/dev/null
            ;;
        consumer_group)
            out="$(op "$key" SNAPSHOT '{}')"
            assert_contains "$out" workers
            ;;
        pubsub)
            out="$(op "$key" PUBLISH '{"channel":"events","value":{"type":"utf8_string","value":"alpha"}}' | json_extract 'x')"
            assert_eq "$out" 0
            ;;
        persistent_topic)
            op "$key" PUBLISH '{"message_key":"m1","value":{"type":"utf8_string","value":"alpha"}}' >/dev/null
            out="$(op "$key" READ '{"from":0,"count":10}')"

            assert_contains "$out" alpha
            ;;
        event_log)
            op "$key" APPEND '{"event_type":"created","aggregate_id":"a1","payload":{"type":"utf8_string","value":"alpha"}}' >/dev/null
            out="$(op "$key" LEN '{}' | json_extract 'x')"

            assert_eq "$out" 1
            ;;
        bitmap)
            op "$key" SET '{"bit":7,"value":true}' >/dev/null

            out="$(op "$key" GETBIT '{"bit":7}' | json_extract 'x')"
            assert_true "$out"

            out="$(op "$key" COUNT '{}' | json_extract 'x')"
            assert_eq "$out" 1
            ;;
        bit_field)
            op "$key" SET '{"offset":4,"width":8,"value":171}' >/dev/null
            out="$(op "$key" GET '{"offset":4,"width":8}' | json_extract 'x')"

            assert_eq "$out" 171
            ;;
        bit_set)
            op "$key" SET '{"bit":7,"value":true}' >/dev/null

            out="$(op "$key" GETBIT '{"bit":7}' | json_extract 'x')"
            assert_true "$out"

            out="$(op "$key" COUNT '{}' | json_extract 'x')"
            assert_eq "$out" 1
            ;;
        roaring_bitmap)
            op "$key" ADD '{"value":42}' >/dev/null

            out="$(op "$key" CONTAINS '{"value":42}' | json_extract 'x')"
            assert_true "$out"

            out="$(op "$key" CARDINALITY '{}' | json_extract 'x')"
            assert_eq "$out" 1
            ;;
        hyperloglog)
            op "$key" ADD '{"value":"alice"}' >/dev/null
            op "$key" ADD '{"value":"bob"}' >/dev/null

            out="$(op "$key" COUNT '{}' | json_extract 'x')"
            [[ "$out" -ge 1 ]] || die "HLL estimate invalid"
            ;;
        bloom_filter)
            op "$key" ADD '{"value":"alice"}' >/dev/null
            out="$(op "$key" CONTAINS '{"value":"alice"}' | json_extract 'x')"

            assert_true "$out"
            ;;
        counting_bloom_filter)
            op "$key" ADD '{"value":"alice"}' >/dev/null
            out="$(op "$key" CONTAINS '{"value":"alice"}' | json_extract 'x')"

            assert_true "$out"
            op "$key" REMOVE '{"value":"alice"}' >/dev/null
            ;;
        cuckoo_filter)
            op "$key" ADD '{"value":"alice"}' >/dev/null
            out="$(op "$key" CONTAINS '{"value":"alice"}' | json_extract 'x')"

            assert_true "$out"
            op "$key" REMOVE '{"value":"alice"}' >/dev/null
            ;;
        count_min_sketch)
            op "$key" ADD '{"value":"alice","count":5}' >/dev/null
            out="$(op "$key" ESTIMATE '{"value":"alice"}' | json_extract 'x')"

            [[ "$out" -ge 5 ]] || die "CMS estimate invalid"
            ;;
        top_k)
            op "$key" ADD '{"value":"alice","count":5}' >/dev/null
            out="$(op "$key" ITEMS '{}')"

            assert_contains "$out" alice
            ;;
        t_digest)
            for n in 1 2 3 4 5; do op "$key" ADD "{\"value\":$n}" >/dev/null; done
            out="$(op "$key" QUANTILE '{"q":0.5}' | json_extract 'x')"

            assert_nonempty "$out"
            ;;
        timeseries)
            local now_ms
            now_ms=$(($(date +%s) * 1000))

            op "$key" ADD "{\"timestamp\":$now_ms,\"value\":10}" >/dev/null
            op "$key" ADD "{\"timestamp\":$((now_ms + 1000)),\"value\":20}" >/dev/null

            out="$(op "$key" LEN '{}' | json_extract 'x')"
            assert_eq "$out" 2
            ;;
        counter_series)
            op "$key" ADD '{"timestamp":1000,"value":5}' >/dev/null
            op "$key" ADD '{"timestamp":1000,"value":7}' >/dev/null

            out="$(op "$key" LAST '{}')"
            assert_contains "$out" 12
            ;;
        gauge_series)
            op "$key" SET '{"timestamp":1000,"value":42.5}' >/dev/null
            out="$(op "$key" LAST '{}')"

            assert_contains "$out" 42.5
            ;;
        histogram)
            op "$key" OBSERVE '{"value":7}' >/dev/null
            out="$(op "$key" SNAPSHOT '{}')"

            assert_contains "$out" count
            ;;
        timeseries_labels)
            op "$key" SET '{"label":"region","value":"ph"}' >/dev/null
            out="$(op "$key" GET '{}')"

            assert_contains "$out" ph
            ;;
        aggregated_series)
            local source="${key}:source"
            create "$source" timeseries '{}'

            op "$source" ADD '{"timestamp":1000,"value":10}' >/dev/null
            op "$source" ADD '{"timestamp":1500,"value":20}' >/dev/null

            out="$(op "$key" AGGREGATE "{\"source_key\":\"$source\",\"from\":0,\"to\":3000,\"bucket_ms\":1000,\"aggregation\":\"avg\"}")"
            assert_contains "$out" 15

            amaquet DEL "{\"keys\":[\"$source\"]}" >/dev/null
            ;;
        geo_point)
            out="$(op "$key" DISTANCE '{"point":{"Lat":14.5995,"Lon":120.9842}}' | json_extract 'x')"
            assert_eq "$out" 0
            ;;
        geo_spatial_index)
            op "$key" ADD '{"id":"manila","point":{"Lat":14.5995,"Lon":120.9842},"metadata":{"country":"PH"}}' >/dev/null
            out="$(op "$key" RADIUS '{"center":{"Lat":14.5995,"Lon":120.9842},"radius_meters":1000,"limit":10}')"

            assert_contains "$out" manila
            ;;
        bounding_box)
            out="$(op "$key" CONTAINS '{"point":{"Lat":14.5,"Lon":121.0}}' | json_extract 'x')"
            assert_true "$out"
            ;;
        polygon)
            out="$(op "$key" CONTAINS '{"point":{"Lat":14.6,"Lon":121.0}}' | json_extract 'x')"
            assert_true "$out"
            ;;
        dense_vector)
            out="$(op "$key" DISTANCE '{"vector":[1,0,0.5],"metric":"cosine"}' | json_extract 'x')"
            python3 -c 'import sys; assert abs(float(sys.argv[1])) < 1e-9' "$out"
            ;;
        sparse_vector)
            out="$(amaquet GET "{\"key\":\"$key\"}")"
            assert_contains "$out" "\"type\": \"$type\""
            ;;
        vector_set)
            op "$key" ADD '{"id":"v1","vector":[1,0,0],"metadata":{"class":"a"}}' >/dev/null
            out="$(op "$key" SEARCH '{"vector":[1,0,0],"k":1}')"

            assert_contains "$out" v1
            ;;
        vector_metadata)
            op "$key" SET '{"field":"lang","value":"en"}' >/dev/null
            out="$(op "$key" GET '{}')"

            assert_contains "$out" en
            ;;
        quantized_vector)
            out="$(op "$key" DEQUANTIZE '{}')"
            assert_contains "$out" 0.1
            ;;
        btree_index)
            op "$key" INSERT '{"value":"b","id":"2"}' >/dev/null
            op "$key" INSERT '{"value":"a","id":"1"}' >/dev/null

            out="$(op "$key" SEARCH '{"value":"a"}')"
            assert_contains "$out" 1
            ;;
        hash_index)
            op "$key" ADD '{"value":"blue","id":"1"}' >/dev/null
            out="$(op "$key" LOOKUP '{"value":"blue"}')"

            assert_contains "$out" 1
            ;;
        radix_tree)
            op "$key" INSERT '{"value":"amaquet","id":"1"}' >/dev/null
            out="$(op "$key" PREFIX '{"prefix":"ama","limit":10}')"

            assert_contains "$out" 1
            ;;
        trie)
            op "$key" INSERT '{"value":"amaquet","id":"1"}' >/dev/null
            out="$(op "$key" PREFIX '{"prefix":"ama","limit":10}')"

            assert_contains "$out" 1
            ;;
        inverted_index)
            op "$key" INDEX '{"id":"d1","text":"amaquet in memory database"}' >/dev/null
            out="$(op "$key" SEARCH '{"query":"memory","limit":10}')"

            assert_contains "$out" d1
            ;;
        hnsw_index)
            op "$key" ADD '{"id":"v1","vector":[1,0,0],"metadata":{}}' >/dev/null
            out="$(op "$key" SEARCH '{"vector":[1,0,0],"k":1}')"

            assert_contains "$out" v1
            ;;
        flat_vector_index)
            op "$key" ADD '{"id":"v1","vector":[1,0,0],"metadata":{}}' >/dev/null
            out="$(op "$key" SEARCH '{"vector":[1,0,0],"k":1}')"

            assert_contains "$out" v1
            ;;
        geospatial_secondary_index)
            op "$key" ADD '{"id":"manila","point":{"Lat":14.5995,"Lon":120.9842},"metadata":{"country":"PH"}}' >/dev/null
            out="$(op "$key" RADIUS '{"center":{"Lat":14.5995,"Lon":120.9842},"radius_meters":1000,"limit":10}')"

            assert_contains "$out" manila
            ;;
        secondary_index)
            op "$key" ADD '{"field":"status","value":"active","id":"u1"}' >/dev/null
            out="$(op "$key" LOOKUP '{"field":"status","value":"active"}')"

            assert_contains "$out" u1
            ;;
        full_text_document)
            op "$key" ADD_TEXT '{"name":"body","value":"hello amaquet","weight":2}' >/dev/null
            op "$key" ADD_TAG '{"name":"tags","values":["go","memory"]}' >/dev/null
            op "$key" ADD_NUMERIC '{"name":"score","value":9.5}' >/dev/null

            out="$(op "$key" SNAPSHOT '{}')"
            assert_contains "$out" amaquet
            ;;
        tag_field)
            op "$key" SET '{"name":"tags","values":["a","b"]}' >/dev/null
            out="$(op "$key" GET '{}')"

            assert_contains "$out" a
            ;;
        numeric_field)
            op "$key" SET '{"name":"score","value":99.5}' >/dev/null
            out="$(op "$key" GET '{}')"

            assert_contains "$out" 99.5
            ;;
        text_field)
            op "$key" SET '{"name":"body","value":"updated","weight":2}' >/dev/null
            out="$(op "$key" GET '{}')"

            assert_contains "$out" updated
            ;;
        adjacency_set)
            op "$key" ADD '{"from":"a","to":"b","weight":1.5}' >/dev/null
            out="$(op "$key" NEIGHBORS '{"from":"a"}')"

            assert_contains "$out" b
            ;;
        atomic_counter)
            out="$(op "$key" ADD '{"delta":5}' | json_extract 'x')"
            assert_eq "$out" 15

            out="$(op "$key" CAS '{"old":15,"new":20}' | json_extract 'x')"
            assert_true "$out"
            ;;
        lease)
            out="$(op "$key" ACQUIRE '{"owner":"tester","token":"t1","ttl_ms":5000}' | json_extract 'x')"
            assert_true "$out"

            out="$(op "$key" HELD '{}' | json_extract 'x')"
            assert_true "$out"

            op "$key" RELEASE '{"token":"t1"}' >/dev/null
            ;;
        lock)
            out="$(op "$key" TRY_LOCK '{"owner":"tester","token":"t1","ttl_ms":5000}' | json_extract 'x')"
            assert_true "$out"

            out="$(op "$key" UNLOCK '{"token":"t1"}' | json_extract 'x')"
            assert_true "$out"
            ;;
        barrier)
            out="$(op "$key" WAIT '{}' | json_extract 'x')"
            assert_true "$out"
            ;;
        semaphore)
            out="$(op "$key" TRY_ACQUIRE '{}' | json_extract 'x')"
            assert_true "$out"

            out="$(op "$key" STATS '{}')"
            assert_contains "$out" limit

            op "$key" RELEASE '{}' >/dev/null
            ;;
        rate_limit_bucket)
            out="$(op "$key" ALLOW '{"cost":1}' | json_extract 'x')"
            assert_true "$out"

            out="$(op "$key" REMAINING '{}' | json_extract 'x')"
            assert_nonempty "$out"
            ;;
        cidr_network)
            out="$(op "$key" CONTAINS '{"ip":"10.1.2.3"}' | json_extract 'x')"
            assert_true "$out"
            ;;
        url)
            out="$(op "$key" HOST '{}' | json_extract 'x')"
            assert_contains "$out" example.com

            out="$(op "$key" SCHEME '{}' | json_extract 'x')"
            assert_eq "$out" https
            ;;
        *) die "missing semantic contract for $type" ;; esac
}

run_type_contract() {
    local type="$1" key="bash:${1}:$$:$RANDOM" out

    start_test_server
    setup_type "$key" "$type"

    common_key_contract "$key" "$type"
    run_type_semantics "$key" "$type"

    out="$(amaquet DEL "{\"keys\":[\"$key\"]}" | json_extract 'x')"
    assert_eq "$out" 1 DEL

    log "PASS $type ($AMAQUET_TEST_PASSES assertions)"
}

fuzz_type() {
    local type="$1" iterations="${AMAQUET_FUZZ_ITERATIONS:-100}" key="fuzz:${1}:$$:$RANDOM" i junk

    start_test_server
    setup_type "$key" "$type"

    for ((i = 0; i < iterations; i++)); do
        junk="FUZZ_$RANDOM$RANDOM"

        assert_command_fails amaquet OP "{\"key\":\"$key\",\"operation\":\"$junk\",\"args\":{\"noise\":$RANDOM}}"
        amaquet TYPE "{\"key\":\"$key\"}" >/dev/null

        if ((i % 10 == 0)); then
            amaquet DEL "{\"keys\":[\"$key\"]}" >/dev/null || true
            setup_type "$key" "$type"
            run_type_semantics "$key" "$type" >/dev/null
        fi
    done

    log "FUZZ PASS $type ($iterations iterations)"
}
