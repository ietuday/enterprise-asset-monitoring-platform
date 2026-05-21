#!/usr/bin/env bash

set -euo pipefail

API_BASE_URL="${API_BASE_URL:-http://localhost:4000}"

echo "Seeding Enterprise Asset Monitoring Platform"
echo "API Base URL: $API_BASE_URL"
echo ""

echo "Creating default users..."

create_user() {
  local name="$1"
  local email="$2"
  local password="$3"
  local role="$4"

  echo "Creating user: $email"

  response=$(curl -s -w "\n%{http_code}" -X POST "$API_BASE_URL/api/auth/register" \
    -H "Content-Type: application/json" \
    -d "{
      \"name\": \"$name\",
      \"email\": \"$email\",
      \"password\": \"$password\",
      \"role\": \"$role\"
    }")

  body=$(echo "$response" | sed '$d')
  status=$(echo "$response" | tail -n1)

  if [[ "$status" == "201" ]]; then
    echo "Created: $email"
  elif [[ "$status" == "409" ]]; then
    echo "Already exists: $email"
  else
    echo "Failed to create $email. Status: $status"
    echo "$body"
  fi
}

create_user "Admin User" "admin@example.com" "admin123" "ADMIN"
create_user "Operator User" "operator@example.com" "operator123" "OPERATOR"
create_user "Viewer User" "viewer@example.com" "viewer123" "VIEWER"

echo ""
echo "Logging in as admin..."

login_response=$(curl -s -X POST "$API_BASE_URL/api/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "admin123"
  }')

TOKEN=$(echo "$login_response" | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')

if [[ -z "$TOKEN" ]]; then
  echo "Failed to login as admin"
  echo "$login_response"
  exit 1
fi

echo "Admin login successful"
echo ""

echo "Creating sample assets..."

create_asset() {
  local id="$1"
  local name="$2"
  local type="$3"
  local location="$4"
  local status="$5"

  echo "Creating asset: $id"

  response=$(curl -s -w "\n%{http_code}" -X POST "$API_BASE_URL/api/assets" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d "{
      \"id\": \"$id\",
      \"name\": \"$name\",
      \"type\": \"$type\",
      \"location\": \"$location\",
      \"status\": \"$status\"
    }")

  body=$(echo "$response" | sed '$d')
  status_code=$(echo "$response" | tail -n1)

  if [[ "$status_code" == "201" ]]; then
    echo "Created asset: $id"
  elif echo "$body" | grep -qi "duplicate\|already exists"; then
    echo "Already exists: $id"
  else
    echo "Asset response status: $status_code"
    echo "$body"
  fi
}

create_asset "motor-101" "Motor 101" "MOTOR" "Pune Factory" "ACTIVE"
create_asset "compressor-101" "Air Compressor 101" "COMPRESSOR" "Mumbai Factory" "ACTIVE"
create_asset "boiler-101" "Boiler Machine 101" "MACHINE" "Pune Factory" "ACTIVE"

echo ""
echo "Sending sample telemetry..."

curl -s -X POST "$API_BASE_URL/api/telemetry" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "assetId": "motor-101",
    "temperature": 70,
    "cpu": 60,
    "memory": 50,
    "status": "RUNNING"
  }' > /dev/null

curl -s -X POST "$API_BASE_URL/api/telemetry" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "assetId": "compressor-101",
    "temperature": 72,
    "cpu": 65,
    "memory": 55,
    "status": "RUNNING"
  }' > /dev/null

curl -s -X POST "$API_BASE_URL/api/telemetry" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "assetId": "boiler-101",
    "temperature": 95,
    "cpu": 70,
    "memory": 60,
    "status": "RUNNING"
  }' > /dev/null

echo "Sample telemetry sent"
echo ""

echo "Seed completed successfully"
echo ""
echo "Default users:"
echo "  ADMIN    admin@example.com / admin123"
echo "  OPERATOR operator@example.com / operator123"
echo "  VIEWER   viewer@example.com / viewer123"
echo ""
echo "Open dashboard:"
echo "  http://localhost:3000"