#!/usr/bin/env bash

set -euo pipefail

# WARNING: This script removes local Docker volumes and resets the E2E environment.
# Use only when you want a clean local run and understand that local DB data will be deleted.

if [[ "$1" != "--yes" ]]; then
  echo "This script will delete Docker volumes and reset the E2E environment."
  echo "Run with: ./scripts/e2e-reset-run.sh --yes"
  exit 1
fi

API_RATE_LIMIT_MAX=10000 \
AUTH_RATE_LIMIT_MAX=1000 \
docker compose down --remove-orphans --volumes

API_RATE_LIMIT_MAX=10000 \
AUTH_RATE_LIMIT_MAX=1000 \
docker compose up -d --build

chmod +x tests/e2e/wait-for-services.sh
./tests/e2e/wait-for-services.sh

chmod +x scripts/seed.sh
./scripts/seed.sh

pushd tests/e2e > /dev/null
npm ci --ignore-scripts
./node_modules/.bin/playwright install --with-deps chromium
npm run test:api:smoke
npm run test:ui
popd > /dev/null
