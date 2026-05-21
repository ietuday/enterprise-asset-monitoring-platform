#!/usr/bin/env bash

set -euo pipefail

API_BASE_URL="${API_BASE_URL:-http://localhost:4000}"
METRICS_BASE_URL="${METRICS_BASE_URL:-http://localhost:5002}"

ASSET_ID="e2e-prom-motor-101"

echo "Running E2E smoke test"
echo "API Base URL: $API_BASE_URL"
echo ""

wait_for_alert_status() {
  local asset_id="$1"
  local expected_status="$2"
  local max_attempts="${3:-20}"
  local sleep_seconds="${4:-3}"

  echo "Waiting for alert status: asset=$asset_id status=$expected_status"

  for attempt in $(seq 1 "$max_attempts"); do
    alerts_response=$(curl -fsS "$API_BASE_URL/api/alerts" \
      -H "Authorization: Bearer $TOKEN")

    if echo "$alerts_response" | grep -q "\"assetId\":\"$asset_id\"" && \
       echo "$alerts_response" | grep -q "\"name\":\"High Temperature\"" && \
       echo "$alerts_response" | grep -q "\"status\":\"$expected_status\""; then
      echo "Found High Temperature alert with status=$expected_status"
      return 0
    fi

    echo "Attempt $attempt/$max_attempts: alert not yet $expected_status"
    sleep "$sleep_seconds"
  done

  echo "Expected alert status not found: $expected_status"
  echo "$alerts_response"
  return 1
}

echo "1. Checking API Gateway health..."
curl -fsS "$API_BASE_URL/health" > /dev/null
echo "API Gateway is healthy"

echo ""
echo "2. Logging in as admin..."

LOGIN_RESPONSE=$(curl -fsS -X POST "$API_BASE_URL/api/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "admin123"
  }')

TOKEN=$(echo "$LOGIN_RESPONSE" | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')

if [[ -z "$TOKEN" ]]; then
  echo "Failed to extract JWT token"
  echo "$LOGIN_RESPONSE"
  exit 1
fi

echo "Login successful"

echo ""
echo "3. Creating test asset..."

CREATE_ASSET_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_BASE_URL/api/assets" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d "{
    \"id\": \"$ASSET_ID\",
    \"name\": \"E2E Prometheus Motor 101\",
    \"type\": \"MOTOR\",
    \"location\": \"Test Factory\",
    \"status\": \"ACTIVE\"
  }")

CREATE_BODY=$(echo "$CREATE_ASSET_RESPONSE" | sed '$d')
CREATE_STATUS=$(echo "$CREATE_ASSET_RESPONSE" | tail -n1)

if [[ "$CREATE_STATUS" == "201" ]]; then
  echo "Asset created"
elif [[ "$CREATE_STATUS" == "409" || "$CREATE_STATUS" == "500" ]]; then
  echo "Asset may already exist, continuing"
else
  echo "Unexpected asset create status: $CREATE_STATUS"
  echo "$CREATE_BODY"
  exit 1
fi

echo ""
echo "4. Sending abnormal telemetry..."

curl -fsS -X POST "$API_BASE_URL/api/telemetry" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d "{
    \"assetId\": \"$ASSET_ID\",
    \"temperature\": 96,
    \"cpu\": 65,
    \"memory\": 50,
    \"status\": \"RUNNING\"
  }" > /dev/null

echo "Abnormal telemetry accepted"

echo ""
echo "5. Verifying Prometheus metric was updated..."

if curl -fsS "$METRICS_BASE_URL/metrics" | grep -q "asset_temperature_celsius{asset_id=\"$ASSET_ID\"} 96"; then
  echo "Temperature metric updated"
else
  echo "Temperature metric was not found or not updated"
  curl -fsS "$METRICS_BASE_URL/metrics" | grep asset_temperature_celsius || true
  exit 1
fi

echo ""
echo "6. Waiting for Alertmanager flow to create OPEN alert..."

wait_for_alert_status "$ASSET_ID" "OPEN" 20 3

echo ""
echo "7. Sending normal telemetry..."

curl -fsS -X POST "$API_BASE_URL/api/telemetry" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d "{
    \"assetId\": \"$ASSET_ID\",
    \"temperature\": 70,
    \"cpu\": 65,
    \"memory\": 50,
    \"status\": \"RUNNING\"
  }" > /dev/null

echo "Normal telemetry accepted"

echo ""
echo "8. Waiting for Alertmanager flow to resolve alert..."

wait_for_alert_status "$ASSET_ID" "RESOLVED" 20 3

echo ""
echo "9. Checking report summary..."

curl -fsS "$API_BASE_URL/api/reports/summary" \
  -H "Authorization: Bearer $TOKEN" > /dev/null

echo "Report summary API is working"

echo ""
echo "E2E smoke test passed successfully"