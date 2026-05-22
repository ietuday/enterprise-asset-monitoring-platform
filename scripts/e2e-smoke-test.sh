#!/usr/bin/env bash

set -euo pipefail

API_BASE_URL="${API_BASE_URL:-http://localhost:4000}"
METRICS_BASE_URL="${METRICS_BASE_URL:-http://localhost:5002}"

RUN_ID="$(date +%s)"

STATIC_ASSET_ID="e2e-prom-motor-${RUN_ID}"
DYNAMIC_ASSET_ID="e2e-dynamic-rule-motor-${RUN_ID}"
DYNAMIC_RULE_NAME="E2E Dynamic High CPU ${RUN_ID}"

echo "Running E2E smoke test"
echo "API Base URL: $API_BASE_URL"
echo "Metrics Base URL: $METRICS_BASE_URL"
echo "Run ID: $RUN_ID"
echo ""

extract_json_field() {
  local json="$1"
  local field="$2"

  python3 - "$json" "$field" <<'PY'
import json
import sys

payload = json.loads(sys.argv[1])
field = sys.argv[2]

value = payload.get(field, "")
print(value)
PY
}

wait_for_alert_status() {
  local asset_id="$1"
  local alert_name="$2"
  local expected_status="$3"
  local max_attempts="${4:-20}"
  local sleep_seconds="${5:-3}"

  echo "Waiting for alert: asset=$asset_id name=$alert_name status=$expected_status"

  for attempt in $(seq 1 "$max_attempts"); do
    alerts_response=$(curl -fsS "$API_BASE_URL/api/alerts" \
      -H "Authorization: Bearer $TOKEN")

    found=$(python3 - "$alerts_response" "$asset_id" "$alert_name" "$expected_status" <<'PY'
import json
import sys

alerts = json.loads(sys.argv[1])
asset_id = sys.argv[2]
alert_name = sys.argv[3]
expected_status = sys.argv[4]

for alert in alerts:
    if (
        alert.get("assetId") == asset_id
        and alert.get("name") == alert_name
        and alert.get("status") == expected_status
    ):
        print("yes")
        sys.exit(0)

print("no")
PY
)

    if [[ "$found" == "yes" ]]; then
      echo "Found alert with expected status: $expected_status"
      return 0
    fi

    echo "Attempt $attempt/$max_attempts: alert not yet $expected_status"
    sleep "$sleep_seconds"
  done

  echo "Expected alert status not found"
  echo "asset=$asset_id"
  echo "name=$alert_name"
  echo "status=$expected_status"
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

TOKEN=$(extract_json_field "$LOGIN_RESPONSE" "token")

if [[ -z "$TOKEN" ]]; then
  echo "Failed to extract JWT token"
  echo "$LOGIN_RESPONSE"
  exit 1
fi

echo "Login successful"

echo ""
echo "3. Creating test asset for static High Temperature flow..."

CREATE_ASSET_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_BASE_URL/api/assets" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d "{
    \"id\": \"$STATIC_ASSET_ID\",
    \"name\": \"E2E Prometheus Motor $RUN_ID\",
    \"type\": \"MOTOR\",
    \"location\": \"Test Factory\",
    \"status\": \"ACTIVE\"
  }")

CREATE_BODY=$(echo "$CREATE_ASSET_RESPONSE" | sed '$d')
CREATE_STATUS=$(echo "$CREATE_ASSET_RESPONSE" | tail -n1)

if [[ "$CREATE_STATUS" == "200" || "$CREATE_STATUS" == "201" ]]; then
  echo "Static test asset created"
elif [[ "$CREATE_STATUS" == "409" || "$CREATE_STATUS" == "500" ]]; then
  echo "Static test asset may already exist, continuing"
else
  echo "Unexpected asset create status: $CREATE_STATUS"
  echo "$CREATE_BODY"
  exit 1
fi

echo ""
echo "4. Sending abnormal temperature telemetry..."

curl -fsS -X POST "$API_BASE_URL/api/telemetry" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d "{
    \"assetId\": \"$STATIC_ASSET_ID\",
    \"temperature\": 96,
    \"cpu\": 65,
    \"memory\": 50,
    \"status\": \"RUNNING\"
  }" > /dev/null

echo "Abnormal temperature telemetry accepted"

echo ""
echo "5. Verifying temperature metric was updated..."

if curl -fsS "$METRICS_BASE_URL/metrics" | grep -q "asset_temperature_celsius{asset_id=\"$STATIC_ASSET_ID\"} 96"; then
  echo "Temperature metric updated"
else
  echo "Temperature metric was not found or not updated"
  curl -fsS "$METRICS_BASE_URL/metrics" | grep asset_temperature_celsius || true
  exit 1
fi

echo ""
echo "6. Waiting for High Temperature alert to become OPEN..."

wait_for_alert_status "$STATIC_ASSET_ID" "High Temperature" "OPEN" 20 3

echo ""
echo "7. Sending normal temperature telemetry..."

curl -fsS -X POST "$API_BASE_URL/api/telemetry" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d "{
    \"assetId\": \"$STATIC_ASSET_ID\",
    \"temperature\": 70,
    \"cpu\": 65,
    \"memory\": 50,
    \"status\": \"RUNNING\"
  }" > /dev/null

echo "Normal temperature telemetry accepted"

echo ""
echo "8. Waiting for High Temperature alert to become RESOLVED..."

wait_for_alert_status "$STATIC_ASSET_ID" "High Temperature" "RESOLVED" 20 3

echo ""
echo "9. Creating dynamic CPU rule..."

RULE_RESPONSE=$(curl -fsS -X POST "$API_BASE_URL/api/rules" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d "{
    \"name\": \"$DYNAMIC_RULE_NAME\",
    \"metric\": \"cpu\",
    \"operator\": \">\",
    \"threshold\": 90,
    \"value\": \"\",
    \"severity\": \"HIGH\",
    \"enabled\": true
  }")

RULE_ID=$(extract_json_field "$RULE_RESPONSE" "id")

if [[ -z "$RULE_ID" ]]; then
  echo "Failed to extract rule ID"
  echo "$RULE_RESPONSE"
  exit 1
fi

echo "Dynamic CPU rule created with ID=$RULE_ID"

echo ""
echo "10. Sending high CPU telemetry..."

curl -fsS -X POST "$API_BASE_URL/api/telemetry" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d "{
    \"assetId\": \"$DYNAMIC_ASSET_ID\",
    \"temperature\": 70,
    \"cpu\": 95,
    \"memory\": 50,
    \"status\": \"RUNNING\"
  }" > /dev/null

echo "High CPU telemetry accepted"

echo ""
echo "11. Verifying CPU metric was updated..."

if curl -fsS "$METRICS_BASE_URL/metrics" | grep -q "asset_cpu_usage_percent{asset_id=\"$DYNAMIC_ASSET_ID\"} 95"; then
  echo "CPU metric updated"
else
  echo "CPU metric was not found or not updated"
  curl -fsS "$METRICS_BASE_URL/metrics" | grep asset_cpu_usage_percent || true
  exit 1
fi

echo ""
echo "12. Waiting for dynamic CPU alert to become OPEN..."

wait_for_alert_status "$DYNAMIC_ASSET_ID" "$DYNAMIC_RULE_NAME" "OPEN" 20 3

echo ""
echo "13. Sending normal CPU telemetry..."

curl -fsS -X POST "$API_BASE_URL/api/telemetry" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d "{
    \"assetId\": \"$DYNAMIC_ASSET_ID\",
    \"temperature\": 70,
    \"cpu\": 50,
    \"memory\": 50,
    \"status\": \"RUNNING\"
  }" > /dev/null

echo "Normal CPU telemetry accepted"

echo ""
echo "14. Waiting for dynamic CPU alert to become RESOLVED..."

wait_for_alert_status "$DYNAMIC_ASSET_ID" "$DYNAMIC_RULE_NAME" "RESOLVED" 20 3

echo ""
echo "15. Checking rule audit history..."

HISTORY_RESPONSE=$(curl -fsS "$API_BASE_URL/api/rules/$RULE_ID/history" \
  -H "Authorization: Bearer $TOKEN")

if echo "$HISTORY_RESPONSE" | grep -q '"action":"CREATED"'; then
  echo "Rule audit history contains CREATED event"
else
  echo "Expected CREATED audit event not found"
  echo "$HISTORY_RESPONSE"
  exit 1
fi

echo ""
echo "16. Checking report summary..."

curl -fsS "$API_BASE_URL/api/reports/summary" \
  -H "Authorization: Bearer $TOKEN" > /dev/null

echo "Report summary API is working"

echo ""
echo "E2E smoke test passed successfully"