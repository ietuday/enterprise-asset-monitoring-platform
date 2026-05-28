#!/usr/bin/env bash

# NOTE:
# This script does NOT remove Docker volumes.
# Database data is preserved.
# Do NOT add --volumes or -v unless you intentionally want to reset the database.

set -e

echo "=========================================="
echo " Reloading Enterprise Asset Monitoring"
echo "=========================================="

SERVICE="${1:-all}"

# All application services.
# Keep postgres/redis/prometheus/alertmanager as infra services.
APP_SERVICES=(
  api-gateway
  auth-service
  asset-service
  telemetry-service
  alert-service
  rule-service
  report-service
  dashboard
  notification-service
)

INFRA_SERVICES=(
  postgres
  redis
  prometheus
  alertmanager
)

ALL_SERVICES=(
  "${INFRA_SERVICES[@]}"
  "${APP_SERVICES[@]}"
)

echo ""
echo "Selected service: $SERVICE"
echo ""

reload_service() {
  local service="$1"

  echo "Reloading $service..."
  docker compose stop "$service" || true
  docker compose rm -f "$service" || true
  docker compose build --no-cache "$service"
  docker compose up -d "$service"
}

service_exists_in_list() {
  local service="$1"
  shift

  for item in "$@"; do
    if [ "$item" = "$service" ]; then
      return 0
    fi
  done

  return 1
}

if [ "$SERVICE" = "all" ]; then
  echo "Stopping all containers..."
  docker compose down --remove-orphans

  echo ""
  echo "Building all services without cache..."
  docker compose build --no-cache

  echo ""
  echo "Starting all services..."
  docker compose up -d

elif [ "$SERVICE" = "app" ]; then
  echo "Reloading all application services..."
  for service in "${APP_SERVICES[@]}"; do
    reload_service "$service"
  done

elif [ "$SERVICE" = "infra" ]; then
  echo "Reloading infrastructure services..."
  for service in "${INFRA_SERVICES[@]}"; do
    reload_service "$service"
  done

elif service_exists_in_list "$SERVICE" "${ALL_SERVICES[@]}"; then
  reload_service "$SERVICE"

else
  echo "Unknown service: $SERVICE"
  echo ""
  echo "Usage:"
  echo "  ./reload.sh"
  echo "  ./reload.sh all"
  echo "  ./reload.sh app"
  echo "  ./reload.sh infra"
  echo "  ./reload.sh api-gateway"
  echo "  ./reload.sh auth-service"
  echo "  ./reload.sh asset-service"
  echo "  ./reload.sh telemetry-service"
  echo "  ./reload.sh alert-service"
  echo "  ./reload.sh rule-service"
  echo "  ./reload.sh report-service"
  echo "  ./reload.sh dashboard"
  echo "  ./reload.sh notification-service"
  echo "  ./reload.sh postgres"
  echo "  ./reload.sh redis"
  echo "  ./reload.sh prometheus"
  echo "  ./reload.sh alertmanager"
  exit 1
fi

echo ""
echo "Container status:"
docker compose ps

echo ""
echo "Health checks:"

echo "API Gateway:"
curl -s http://localhost:4000/health || true
echo ""

echo "Dashboard:"
curl -s -I http://localhost:3000 | head -n 1 || true
echo ""

echo ""
echo "Reload completed."
echo "Open: http://localhost:3000"
echo "Hard refresh browser: Ctrl + Shift + R"