#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/scenario.sh"

scenario_init
g="$(scenario_key geo)"
v="$(scenario_key vec)"
h="$(scenario_key hnsw)"

create "$g" geo_spatial_index '{}'
create "$v" vector_set '{"dim":3,"metric":"cosine"}'
create "$h" hnsw_index '{"dim":3,"metric":"cosine","m":8,"ef_search":32}'

op "$g" ADD '{"id":"manila","point":{"Lat":14.5995,"Lon":120.9842},"metadata":{"city":"Manila"}}' >/dev/null
op "$g" ADD '{"id":"makati","point":{"Lat":14.5547,"Lon":121.0244},"metadata":{"city":"Makati"}}' >/dev/null

assert_contains "$(op "$g" RADIUS '{"center":{"Lat":14.5995,"Lon":120.9842},"radius_meters":10000,"limit":10}')" manila
for ((i = 0; i < AMAQUET_SCENARIO_ITERATIONS; i++)); do
    x=$(python3 -c "print(($i+1)/100.0)")
    op "$v" ADD "{\"id\":\"v$i\",\"vector\":[$x,1,0],\"metadata\":{\"i\":$i}}" >/dev/null
    op "$h" ADD "{\"id\":\"h$i\",\"vector\":[$x,1,0],\"metadata\":{\"i\":$i}}" >/dev/null
done

assert_nonempty "$(op "$v" SEARCH '{"vector":[0,1,0],"k":5}' | json_len)"
assert_nonempty "$(op "$h" SEARCH '{"vector":[0,1,0],"k":5}' | json_len)"
server_healthy

echo "PASS geo/vector workloads"
