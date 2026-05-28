#!/usr/bin/env bash

set -euo pipefail

API_BASE_URL="${API_BASE_URL:-http://localhost:4000}"
DASHBOARD_URL="${DASHBOARD_URL:-http://localhost:3000}"
TIMEOUT_SECONDS="${E2E_WAIT_TIMEOUT_SECONDS:-120}"

wait_for_url() {
  local name="$1"
  shift
  local urls=("$@")
  local started_at
  started_at=$(date +%s)

  echo "Waiting for ${name}..."

  while true; do
    for url in "${urls[@]}"; do
      if curl -fsS "$url" >/dev/null 2>&1; then
        echo "${name} is ready at ${url}"
        return 0
      fi
    done

    if (( $(date +%s) - started_at >= TIMEOUT_SECONDS )); then
      echo "Timed out waiting for ${name}"
      echo "Tried URLs: ${urls[*]}"
      return 1
    fi

    sleep 2
  done
}

wait_for_url "API Gateway" "${API_BASE_URL}/health" "${API_BASE_URL}/api/health"
wait_for_url "Dashboard" "${DASHBOARD_URL}"

echo "E2E service readiness checks passed"
