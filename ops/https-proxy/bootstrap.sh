#!/usr/bin/env bash
set -Eeuo pipefail

require_var() {
  local name="$1"
  if [ -z "${!name:-}" ]; then
    echo "Missing required env var: $name"
    exit 1
  fi
}

require_var FRONTEND_DOMAIN
require_var BACKEND_DOMAIN
require_var TRACKING_DOMAIN
require_var RECOGNIZING_DOMAIN

if ! command -v docker >/dev/null 2>&1; then
  echo "docker is required but not installed"
  exit 1
fi

PROXY_DIR="${PROXY_DIR:-/opt/fizon/proxy}"
FRONTEND_UPSTREAM_PORT="${FRONTEND_UPSTREAM_PORT:-18081}"
BACKEND_UPSTREAM_PORT="${BACKEND_UPSTREAM_PORT:-18080}"
TRACKING_UPSTREAM_PORT="${TRACKING_UPSTREAM_PORT:-18082}"
RECOGNIZING_UPSTREAM_PORT="${RECOGNIZING_UPSTREAM_PORT:-18083}"

mkdir -p "${PROXY_DIR}"
cp "$(dirname "$0")/docker-compose.yml" "${PROXY_DIR}/docker-compose.yml"

if [ -n "${LETSENCRYPT_EMAIL:-}" ]; then
  cat > "${PROXY_DIR}/Caddyfile" <<EOCFG
{
  email ${LETSENCRYPT_EMAIL}
}

${FRONTEND_DOMAIN} {
  encode gzip zstd
  reverse_proxy 127.0.0.1:${FRONTEND_UPSTREAM_PORT}
}

${BACKEND_DOMAIN} {
  encode gzip zstd
  reverse_proxy 127.0.0.1:${BACKEND_UPSTREAM_PORT}
}

${TRACKING_DOMAIN} {
  encode gzip zstd
  reverse_proxy 127.0.0.1:${TRACKING_UPSTREAM_PORT}
}

${RECOGNIZING_DOMAIN} {
  encode gzip zstd
  reverse_proxy 127.0.0.1:${RECOGNIZING_UPSTREAM_PORT}
}
EOCFG
else
  cat > "${PROXY_DIR}/Caddyfile" <<EOCFG
${FRONTEND_DOMAIN} {
  encode gzip zstd
  reverse_proxy 127.0.0.1:${FRONTEND_UPSTREAM_PORT}
}

${BACKEND_DOMAIN} {
  encode gzip zstd
  reverse_proxy 127.0.0.1:${BACKEND_UPSTREAM_PORT}
}

${TRACKING_DOMAIN} {
  encode gzip zstd
  reverse_proxy 127.0.0.1:${TRACKING_UPSTREAM_PORT}
}

${RECOGNIZING_DOMAIN} {
  encode gzip zstd
  reverse_proxy 127.0.0.1:${RECOGNIZING_UPSTREAM_PORT}
}
EOCFG
fi

cd "${PROXY_DIR}"
docker compose up -d

echo "HTTPS proxy is up."
echo "Proxy config: ${PROXY_DIR}/Caddyfile"
