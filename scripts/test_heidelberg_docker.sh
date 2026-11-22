#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
image="${1:?Pass the local test image tag}"
run_id="heidelberg-safety1-$(date +%s)-$$"
network=""
sim=""
app=""
cleanup() {
  result=$?
  if [[ "$result" != 0 && -n "$app" ]]; then docker logs "$app" || true; fi
  if [[ -n "$app" ]]; then docker rm -f "$app" >/dev/null; fi
  if [[ -n "$sim" ]]; then docker rm -f "$sim" >/dev/null; fi
  if [[ -n "$network" ]]; then docker network rm "$network" >/dev/null; fi
  exit "$result"
}
trap cleanup EXIT
network="$(docker network create --internal --label "heidelberg-test=$run_id" "$run_id")"
sim="$(docker run -d --network "$network" --network-alias sim --label "heidelberg-test=$run_id" \
  --mount "type=bind,src=$root,dst=/test,readonly" --entrypoint python3 \
  python:3.12-alpine@sha256:c5ba3deef00c7758561a0f03c26f44f10589263d40e732b8748e0d1f38f6dd89 /test/scripts/heidelberg_dual_runtime.py)"
app="$(docker run -d --network "$network" --network-alias evcc --label "heidelberg-test=$run_id" \
  -e EVCC_NETWORK_PORT=7071 --mount "type=bind,src=$root/tests,dst=/test,readonly" \
  --entrypoint evcc "$image" --config /test/heidelberg-dual-runtime.evcc.yaml \
  --database /tmp/heidelberg-test.db --disable-auth)"
docker exec "$sim" python3 /test/scripts/heidelberg_dual_runtime.py --check http://evcc:7071
docker exec "$app" wget -qO /dev/null http://localhost:7071
test "$(docker image inspect --format '{{.Os}}/{{.Architecture}}' "$image")" = linux/amd64
echo "PASS: isolated Docker application and configurable-port healthcheck"
