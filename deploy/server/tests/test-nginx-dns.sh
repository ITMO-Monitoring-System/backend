#!/usr/bin/env bash
set -Eeuo pipefail

config="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/nginx.conf}"
test_dir="$(mktemp -d)"
test_id="itmo-nginx-test-$$"
network="$test_id"
gateway="$test_id-gateway"
backend_old="$test_id-backend-old"
backend_new="$test_id-backend-new"
tracking_old="$test_id-tracking-old"
tracking_new="$test_id-tracking-new"

cleanup() {
  docker rm -f "$gateway" "$backend_old" "$backend_new" "$tracking_old" "$tracking_new" >/dev/null 2>&1 || true
  docker network rm "$network" >/dev/null 2>&1 || true
  rm -rf "$test_dir"
}
trap cleanup EXIT

mkdir -p "$test_dir/backend-old/api" "$test_dir/backend-new/api" \
  "$test_dir/tracking-old" "$test_dir/tracking-new"
printf 'backend-old\n' > "$test_dir/backend-old/api/health"
printf 'backend-new\n' > "$test_dir/backend-new/api/health"
printf 'tracking-old\n' > "$test_dir/tracking-old/health"
printf 'tracking-new\n' > "$test_dir/tracking-new/health"

docker network create "$network" >/dev/null
docker run -d --network "$network" --network-alias backend --name "$backend_old" \
  -v "$test_dir/backend-old:/srv:ro" python:3.12-alpine \
  python -m http.server 8080 --bind 0.0.0.0 --directory /srv >/dev/null
docker run -d --network "$network" --network-alias face_tracking --name "$tracking_old" \
  -v "$test_dir/tracking-old:/srv:ro" python:3.12-alpine \
  python -m http.server 8000 --bind 0.0.0.0 --directory /srv >/dev/null
docker run -d --network "$network" --name "$gateway" \
  -v "$config:/etc/nginx/conf.d/default.conf:ro" nginx:alpine >/dev/null
docker exec "$gateway" nginx -t

await_response() {
  local endpoint="$1" expected="$2" response
  for (( attempt=0; attempt<30; attempt++ )); do
    if response="$(docker exec "$gateway" wget -q -T 2 -O - "http://127.0.0.1$endpoint" 2>/dev/null)" \
      && [[ "$response" == "$expected" ]]; then
      return 0
    fi
    sleep 1
  done
  echo "Expected $expected from $endpoint, got ${response:-no response}" >&2
  return 1
}

await_response /api/health backend-old
await_response /tracking/health tracking-old

# Start replacements before removing the old containers so Docker assigns new IPs.
docker run -d --network "$network" --network-alias backend --name "$backend_new" \
  -v "$test_dir/backend-new:/srv:ro" python:3.12-alpine \
  python -m http.server 8080 --bind 0.0.0.0 --directory /srv >/dev/null
docker run -d --network "$network" --network-alias face_tracking --name "$tracking_new" \
  -v "$test_dir/tracking-new:/srv:ro" python:3.12-alpine \
  python -m http.server 8000 --bind 0.0.0.0 --directory /srv >/dev/null
docker rm -f "$backend_old" "$tracking_old" >/dev/null

await_response /api/health backend-new
await_response /tracking/health tracking-new

# A working HTML page must not be mistaken for a healthy API proxy.
docker rm -f "$backend_new" >/dev/null
docker exec "$gateway" wget -q -T 2 -O /dev/null http://127.0.0.1/fizon/
if docker exec "$gateway" wget -q -T 2 -O /dev/null http://127.0.0.1/api/health; then
  echo 'API probe unexpectedly succeeded after backend stopped' >&2
  exit 1
fi

echo 'Nginx follows replacement container IPs; API probe detects upstream failure.'
