#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ENV_FILE="${1:-$ROOT_DIR/deploy/.env.host-dev}"
DATA_DIR_DEFAULT="$ROOT_DIR/deploy/dev-data"
SERVER_PID=""

if [[ ! -f "$ENV_FILE" ]]; then
  echo "missing env file: $ENV_FILE" >&2
  exit 1
fi

checksum_tree() {
  {
    find "$ROOT_DIR/backend" -type f \
      \( -name '*.go' -o -name '*.sql' -o -name 'go.mod' -o -name 'go.sum' \) \
      ! -path '*/bin/*' -print0
    find "$ROOT_DIR/deploy" -maxdepth 1 -type f \
      \( -name '.env.host-dev' -o -name '.env.host-dev.example' \) -print0 2>/dev/null || true
  } | sort -z | xargs -0 sha256sum 2>/dev/null | sha256sum | awk '{print $1}'
}

stop_server() {
  if [[ -n "${SERVER_PID}" ]] && kill -0 "${SERVER_PID}" 2>/dev/null; then
    kill "${SERVER_PID}" 2>/dev/null || true
    wait "${SERVER_PID}" 2>/dev/null || true
  fi
  SERVER_PID=""
}

start_server() {
  echo "[backend-watch] starting Go server on http://127.0.0.1:8080"
  (
    set -a
    # shellcheck disable=SC1090
    source "$ENV_FILE"
    set +a

    export DATA_DIR="${DATA_DIR:-$DATA_DIR_DEFAULT}"
    export AUTO_SETUP="${AUTO_SETUP:-true}"
    export SERVER_HOST="${SERVER_HOST:-0.0.0.0}"
    export SERVER_PORT="${SERVER_PORT:-8080}"
    export SERVER_MODE="${SERVER_MODE:-debug}"

    mkdir -p "$DATA_DIR"
    cd "$ROOT_DIR/backend"
    exec go run ./cmd/server
  ) &
  SERVER_PID=$!
}

cleanup() {
  stop_server
}

trap cleanup EXIT INT TERM

echo "[backend-watch] env file: $ENV_FILE"
echo "[backend-watch] data dir: ${DATA_DIR_DEFAULT}"
start_server
LAST_CHECKSUM="$(checksum_tree)"

while true; do
  sleep 1
  CURRENT_CHECKSUM="$(checksum_tree)"
  if [[ "$CURRENT_CHECKSUM" != "$LAST_CHECKSUM" ]]; then
    echo "[backend-watch] change detected, restarting..."
    stop_server
    start_server
    LAST_CHECKSUM="$CURRENT_CHECKSUM"
  fi
done
