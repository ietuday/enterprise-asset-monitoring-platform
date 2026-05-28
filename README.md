# Enterprise Asset Monitoring Platform

![CI](https://github.com/ietuday/enterprise-asset-monitoring-platform/actions/workflows/ci.yml/badge.svg)


An enterprise-grade microservices platform for monitoring assets, collecting telemetry, evaluating dynamic rules, generating alerts, producing reports, and visualizing system health.

This project demonstrates a production-style backend architecture using Go, Node.js, Python FastAPI, React, PostgreSQL, Prometheus, Alertmanager, Grafana, JWT authentication, RBAC, audit logging, Docker Compose, and dynamic monitoring rules.

---

## 1. Project Overview

The platform monitors enterprise assets such as machines, devices, containers, or applications.

It supports:

- Asset management
- Telemetry ingestion
- Dynamic monitoring rules
- Prometheus rule generation
- Prometheus-based alert evaluation
- Alertmanager webhook delivery
- Notification channel management and alert delivery
- SLA tracking and escalation workflows
- Alert lifecycle management
- Alert deduplication
- Auto alert resolution
- JWT authentication
- Role-based access control
- Audit logging
- Reporting APIs
- React dashboard
- Prometheus metrics
- Grafana dashboards
- Dockerized local deployment

---

## 2. Architecture

```text
React Dashboard
      |
      v
Node.js API Gateway
      |
      |-- Auth Service       Node.js
      |-- Asset Service      Go
      |-- Telemetry Service  Go
      |-- Alert Service      Go
      |-- Rule Service       Go
      |-- Report Service     Python FastAPI
      |-- Notification Svc   Go
      |
      v
PostgreSQL

Monitoring / Alerting:
Telemetry Service -> /metrics -> Prometheus -> Alertmanager -> Alert Service

Observability:
Go Services -> /metrics -> Prometheus -> Grafana
```

For detailed architecture diagrams, see [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

For Dynamic Monitoring Rules, see [docs/DYNAMIC-RULES.md](docs/DYNAMIC-RULES.md).

For API testing commands, see [docs/API-TESTING.md](docs/API-TESTING.md).

---

## 3. Services

| Service | Tech | Port | Responsibility |
|---|---|---:|---|
| Dashboard | React | 3000 | Frontend UI |
| API Gateway | Node.js | 4000 | Routing, JWT validation, RBAC, audit logging |
| Auth Service | Node.js | 4001 | Login, register, JWT generation |
| Asset Service | Go | 5001 | Asset CRUD |
| Telemetry Service | Go | 5002 | Telemetry ingestion and Prometheus metric exposure |
| Alert Service | Go | 5003 | Alert lifecycle, incident management, SLA tracking, escalation |
| Rule Service | Go | 5004 | Dynamic rule CRUD and Prometheus rule generation |
| Report Service | Python FastAPI | 8000 | Reports and summary APIs |
| Notification Service | Go | 8090 | Email/webhook channels, delivery history, retries |
| Prometheus | Prometheus | 9090 | Metrics scraping and alert rule evaluation |
| Alertmanager | Alertmanager | 9093 | Alert routing and webhook delivery |
| Grafana | Grafana | 3001 | Metrics visualization |
| PostgreSQL | PostgreSQL | 5435 | Database |
| Redis | Redis | 6379 | Cache-ready infrastructure |

---

## 4. Main Flow

```text
1. User logs in through React dashboard or API.
2. Auth Service returns JWT.
3. API Gateway validates JWT.
4. API Gateway applies RBAC.
5. User creates or views assets.
6. Telemetry Service receives asset telemetry.
7. Telemetry Service stores telemetry in PostgreSQL.
8. Telemetry Service exposes latest telemetry as Prometheus metrics.
9. Prometheus scrapes telemetry metrics.
10. Prometheus evaluates static and dynamic alert rules.
11. Alertmanager receives firing/resolved alerts.
12. Alertmanager sends webhook to Alert Service.
13. Alert Service stores alert lifecycle in PostgreSQL.
14. Alert Service sends notification-worthy events to Notification Service.
15. Alert Service tracks SLA deadlines and records escalation history.
16. Notification Service delivers email/webhook notifications and records delivery history.
17. Report Service returns dashboard summaries.
18. Grafana visualizes runtime and business metrics.
```

---

## 5. Roles and Permissions

| Role | Permissions |
|---|---|
| ADMIN | Full access |
| OPERATOR | View data, submit telemetry, acknowledge/resolve alerts |
| VIEWER | Read-only access to assets, alerts, telemetry, reports, and rules |

Examples:

```text
VIEWER can read assets but cannot create assets.
OPERATOR can submit telemetry but cannot create assets.
ADMIN can create assets and manage rules.
```

---

## 6. Ports

| Component | URL |
|---|---|
| React Dashboard | http://localhost:3000 |
| API Gateway | http://localhost:4000 |
| Auth Service | http://localhost:4001 |
| Asset Service | http://localhost:5001 |
| Telemetry Service | http://localhost:5002 |
| Alert Service | http://localhost:5003 |
| Notification Service | http://localhost:8090 |
| Rule Service | http://localhost:5004 |
| Report Service | http://localhost:8000 |
| Prometheus | http://localhost:9090 |
| Alertmanager | http://localhost:9093 |
| Grafana | http://localhost:3001 |

---

## 7. Run Project

Start all services:

```bash
docker compose up -d --build
```

Check containers:

```bash
docker ps
```

Stop all services:

```bash
docker compose down
```

Stop all services and remove local volumes:

```bash
docker compose down -v
```

---

## 8. Makefile Commands

Useful project commands:

```bash
make help
make build
make up
make down
make restart
make logs
make ps
make health
make test
make test-api
make test-metrics
make seed
make e2e
make db-shell
```

Recommended demo flow:

```bash
make build
make seed
make e2e
```

---

## 9. Environment Configuration

The project uses a root `.env` file.

Example values are provided in:

```text
.env.example
```

Important environment variables:

```env
POSTGRES_USER=monitoring_user
POSTGRES_PASSWORD=monitoring_pass
POSTGRES_DB=monitoring_db

JWT_SECRET=change-this-secret

API_GATEWAY_PORT=4000
AUTH_SERVICE_PORT=4001
ASSET_SERVICE_PORT=5001
TELEMETRY_SERVICE_PORT=5002
ALERT_SERVICE_PORT=5003
NOTIFICATION_SERVICE_PORT=8090
RULE_SERVICE_PORT=5004
REPORT_SERVICE_PORT=8000
DASHBOARD_PORT=3000
GRAFANA_PORT=3001
PROMETHEUS_PORT=9090

AUTH_SERVICE_URL=http://auth-service:4001
ASSET_SERVICE_URL=http://asset-service:5001
TELEMETRY_SERVICE_URL=http://telemetry-service:5002
ALERT_SERVICE_URL=http://alert-service:5003
NOTIFICATION_SERVICE_URL=http://notification-service:8090
RULE_SERVICE_URL=http://rule-service:5004
REPORT_SERVICE_URL=http://report-service:8000

SMTP_HOST=
SMTP_PORT=
SMTP_USER=
SMTP_PASSWORD=
SMTP_FROM=

SLA_WORKER_ENABLED=true
SLA_CHECK_INTERVAL_SECONDS=60

ENABLE_DIRECT_ALERTING=false

PROMETHEUS_RULES_FILE=/etc/prometheus/rules/dynamic-rules.yml
PROMETHEUS_RELOAD_URL=http://prometheus:9090/-/reload
```

Do not commit the real `.env` file.

Notification email delivery requires `SMTP_HOST` and `SMTP_FROM`. If SMTP is not configured, email delivery fails gracefully and the failed attempt is recorded in notification history.

SLA tracking is owned by Alert Service. Create policies through `/api/sla-policies`; incidents receive acknowledgement and resolution deadlines based on severity. The SLA worker runs every `SLA_CHECK_INTERVAL_SECONDS` seconds unless `SLA_WORKER_ENABLED=false`.

Create a webhook notification channel through the API Gateway:

```bash
curl -X POST http://localhost:4000/api/notification-channels \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Local Webhook",
    "type": "WEBHOOK",
    "target": "http://host.docker.internal:9000/webhook",
    "enabled": true
  }'
```

Create an email notification channel:

```json
{
  "name": "Ops Email",
  "type": "EMAIL",
  "target": "ops@example.com",
  "enabled": true
}
```

---

## 10. Default Users

Create users using Auth API or run:

```bash
make seed
```

### Admin

```json
{
  "name": "Admin User",
  "email": "admin@example.com",
  "password": "admin123",
  "role": "ADMIN"
}
```

### Operator

```json
{
  "name": "Operator User",
  "email": "operator@example.com",
  "password": "operator123",
  "role": "OPERATOR"
}
```

### Viewer

```json
{
  "name": "Viewer User",
  "email": "viewer@example.com",
  "password": "viewer123",
  "role": "VIEWER"
}
```

---

## 11. API Examples

### Login

```bash
curl -X POST http://localhost:4000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "admin123"
  }'
```

Set token:

```bash
TOKEN="paste-token-here"
```

### Create Asset

```bash
curl -X POST http://localhost:4000/api/assets \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "id": "motor-101",
    "name": "Motor 101",
    "type": "MOTOR",
    "location": "Pune Factory",
    "status": "ACTIVE"
  }'
```

### List Assets

```bash
curl http://localhost:4000/api/assets \
  -H "Authorization: Bearer $TOKEN"
```

### Send Telemetry

```bash
curl -X POST http://localhost:4000/api/telemetry \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "assetId": "motor-101",
    "temperature": 96,
    "cpu": 65,
    "memory": 50,
    "status": "RUNNING"
  }'
```

Telemetry Service stores the telemetry and exposes metrics such as:

```text
asset_temperature_celsius
asset_cpu_usage_percent
asset_memory_usage_percent
telemetry_received_total
```

### List Alerts

```bash
curl http://localhost:4000/api/alerts \
  -H "Authorization: Bearer $TOKEN"
```

### Report Summary

```bash
curl http://localhost:4000/api/reports/summary \
  -H "Authorization: Bearer $TOKEN"
```

Example response:

```json
{
  "totalAssets": 2,
  "totalAlerts": 10,
  "openAlerts": 1,
  "acknowledgedAlerts": 0,
  "resolvedAlerts": 9,
  "criticalAlerts": 8,
  "highAlerts": 2
}
```

---

## 12. Alerting Modes

The platform supports two alerting modes.

### Direct Alerting

```env
ENABLE_DIRECT_ALERTING=true
```

In this mode, Telemetry Service evaluates telemetry rules internally and directly calls Alert Service to create or resolve alerts.

### Prometheus + Alertmanager Alerting

```env
ENABLE_DIRECT_ALERTING=false
```

In this mode, Telemetry Service stores telemetry and exposes Prometheus metrics. Prometheus evaluates alert rules, sends alerts to Alertmanager, and Alertmanager forwards firing/resolved alerts to Alert Service through a webhook.

This is the recommended mode and is closer to the `lems-monitoring` architecture.

---

## 13. Static Alert Rules

Static Prometheus rules are stored in:

```text
infra/prometheus/rules/alert-rules.yml
```

Example static rule:

```yaml
groups:
  - name: telemetry-alert-rules
    interval: 5s
    rules:
      - alert: HighTemperature
        expr: asset_temperature_celsius > 80
        for: 5s
        labels:
          severity: critical
        annotations:
          summary: "High temperature detected"
          description: "Asset {{ $labels.asset_id }} temperature is above threshold"
          asset_id: "{{ $labels.asset_id }}"
          alert_name: "High Temperature"
```

---

## 14. Dynamic Monitoring Rules

Dynamic Monitoring Rules allow thresholds to be managed from APIs instead of being hardcoded.

Rule flow:

```text
ADMIN creates rule through /api/rules
        |
        v
Rule Service stores rule in PostgreSQL
        |
        v
Rule Service generates dynamic-rules.yml
        |
        v
Rule Service calls Prometheus reload API
        |
        v
Prometheus evaluates the new rule
        |
        v
Alertmanager sends webhook to Alert Service
```

### Rule Model

```json
{
  "name": "Dynamic High CPU",
  "metric": "cpu",
  "operator": ">",
  "threshold": 90,
  "severity": "HIGH",
  "enabled": true
}
```

Supported metrics:

```text
temperature
cpu
memory
```

Metric mapping:

| Rule Metric | Prometheus Metric |
|---|---|
| temperature | asset_temperature_celsius |
| cpu | asset_cpu_usage_percent |
| memory | asset_memory_usage_percent |

Supported operators:

```text
>
>=
<
<=
==
!=
```

### Create Dynamic Rule

```bash
curl -X POST http://localhost:4000/api/rules \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "Dynamic High CPU",
    "metric": "cpu",
    "operator": ">",
    "threshold": 90,
    "severity": "HIGH",
    "enabled": true
  }'
```

### List Rules

```bash
curl http://localhost:4000/api/rules \
  -H "Authorization: Bearer $TOKEN"
```

### List Enabled Rules

```bash
curl http://localhost:4000/api/rules/enabled \
  -H "Authorization: Bearer $TOKEN"
```

### Update Rule

```bash
curl -X PUT http://localhost:4000/api/rules/1 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "Dynamic High CPU",
    "metric": "cpu",
    "operator": ">",
    "threshold": 85,
    "severity": "HIGH",
    "enabled": true
  }'
```

### Delete Rule

```bash
curl -X DELETE http://localhost:4000/api/rules/1 \
  -H "Authorization: Bearer $TOKEN"
```

### Generated Prometheus Rule

A dynamic rule like:

```json
{
  "name": "Dynamic High CPU",
  "metric": "cpu",
  "operator": ">",
  "threshold": 90,
  "severity": "HIGH",
  "enabled": true
}
```

generates:

```yaml
groups:
  - name: dynamic-monitoring-rules
    interval: 5s
    rules:
      - alert: DynamicHighCPU
        expr: asset_cpu_usage_percent > 90.00
        for: 5s
        labels:
          severity: high
        annotations:
          summary: "Dynamic High CPU"
          description: "Dynamic rule Dynamic High CPU triggered for asset {{ $labels.asset_id }}"
          asset_id: "{{ $labels.asset_id }}"
          alert_name: "Dynamic High CPU"
```

Generated file:

```text
infra/prometheus/rules/dynamic-rules.yml
```

---

## 15. Test Dynamic Rule Flow

Create rule:

```bash
curl -X POST http://localhost:4000/api/rules \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "Dynamic High CPU",
    "metric": "cpu",
    "operator": ">",
    "threshold": 90,
    "severity": "HIGH",
    "enabled": true
  }'
```

Send telemetry:

```bash
curl -X POST http://localhost:4000/api/telemetry \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "assetId": "dynamic-prom-motor-101",
    "temperature": 70,
    "cpu": 95,
    "memory": 50,
    "status": "RUNNING"
  }'
```

Check metric:

```bash
curl http://localhost:5002/metrics | grep asset_cpu_usage_percent
```

Check Prometheus:

```text
http://localhost:9090/alerts
```

Check Alertmanager:

```text
http://localhost:9093
```

Check app alerts:

```bash
curl http://localhost:4000/api/alerts \
  -H "Authorization: Bearer $TOKEN"
```

Expected:

```text
dynamic-prom-motor-101 Dynamic High CPU OPEN
```

Resolve dynamic alert:

```bash
curl -X POST http://localhost:4000/api/telemetry \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "assetId": "dynamic-prom-motor-101",
    "temperature": 70,
    "cpu": 50,
    "memory": 50,
    "status": "RUNNING"
  }'
```

Expected:

```text
dynamic-prom-motor-101 Dynamic High CPU RESOLVED
```

---

## 16. Alert Lifecycle

```text
OPEN -> ACKNOWLEDGED -> RESOLVED
```

The platform supports:

- Alert creation
- Alert deduplication
- Alert acknowledgement
- Alert resolution
- Auto-resolution when telemetry becomes normal
- Alertmanager firing/resolved webhook processing

---

## 17. Alert Deduplication

Duplicate alerts are prevented using:

```text
asset_id + alert_name + status in OPEN / ACKNOWLEDGED
```

If an active alert already exists for the same asset and alert type, the system returns the existing alert instead of creating a duplicate.

---

## 18. Audit Logging

API Gateway records audit logs in PostgreSQL.

Captured fields:

- user id
- user email
- user role
- HTTP method
- path
- status code
- action
- timestamp

Check audit logs:

```bash
docker exec -it monitoring-postgres psql -U monitoring_user -d monitoring_db
```

```sql
SELECT id, user_email, user_role, method, path, status_code, action, created_at
FROM audit_logs
ORDER BY created_at DESC
LIMIT 10;
```

---

## 19. Prometheus Metrics

Go services expose metrics:

```text
Asset Service      /metrics
Telemetry Service  /metrics
Alert Service      /metrics
Alertmanager       /metrics
```

Prometheus URL:

```text
http://localhost:9090
```

Useful queries:

```promql
up
```

```promql
go_goroutines
```

```promql
telemetry_received_total
```

```promql
asset_temperature_celsius
```

```promql
asset_cpu_usage_percent
```

```promql
asset_memory_usage_percent
```

```promql
alerts_created_total
```

```promql
alerts_resolved_total
```

---

## 20. Grafana

Grafana URL:

```text
http://localhost:3001
```

Default login:

```text
admin / admin
```

Prometheus data source:

```text
http://prometheus:9090
```

Recommended panels:

| Panel | Query |
|---|---|
| Service Availability | `up` |
| Total Telemetry Received | `sum(telemetry_received_total)` |
| Temperature by Asset | `asset_temperature_celsius` |
| CPU by Asset | `asset_cpu_usage_percent` |
| Memory by Asset | `asset_memory_usage_percent` |
| Alerts Created by Severity | `sum(alerts_created_total) by (severity)` |
| Alerts Resolved by Asset | `sum(alerts_resolved_total) by (asset_id)` |

---

## 21. Testing

Run Go unit tests:

```bash
make test
```

Run API health checks:

```bash
make test-api
```

Run metrics checks:

```bash
make test-metrics
```

Run end-to-end smoke test:

```bash
make e2e
```

The E2E test validates:

- Admin login
- Asset creation
- Telemetry ingestion
- Prometheus metric update
- Alertmanager-based alert creation
- Alert auto-resolution
- Report summary API

### Automated E2E Testing

The `tests/e2e` suite is now UI-driven by default. The Playwright UI flow exercises the dashboard user experience and validates API Gateway and backend services through real user interactions.

Default smoke tests:

- `npm run test:api:smoke` — minimal API smoke validation for auth and notification delivery flow
- `npm run test:ui` — full platform flow through the dashboard UI

Deep API tests are available separately:

- `npm run test:api:deep`

Prerequisites:

- Docker
- Docker Compose
- Node.js 20.19+
- npm

Run locally:

```bash
docker compose up -d --build
./tests/e2e/wait-for-services.sh
./scripts/seed.sh

cd tests/e2e
npm ci
npx playwright install --with-deps chromium
npm run test:api:smoke
npm run test:ui
```

Or run both suites:

```bash
npm test
```

For a clean local reset and validation, use `./scripts/e2e-reset-run.sh`, but note this deletes local Docker volumes.

The UI-driven E2E flow covers login, rule and notification channel creation, telemetry ingestion, incident visibility, notifications history, and SLA page validation. API smoke tests keep the gateway stable, while deep API polling tests are kept out of the default CI path.

---

## 22. Tech Stack

### Backend

- Go
- Node.js
- Python FastAPI

### Frontend

- React
- Vite
- Axios

### Database and Infrastructure

- PostgreSQL
- Redis
- Docker Compose

### Observability and Alerting

- Prometheus
- Alertmanager
- Grafana

### Security

- JWT authentication
- RBAC
- Audit logging

---

## 23. SonarQube CI Analysis

The repository includes a SonarQube/SonarCloud workflow at `.github/workflows/sonarqube.yml`.
It runs on pushes and pull requests targeting `master` and `develop`, generates Go and dashboard coverage, and scans `services`, `web`, and `scripts`.

Configure these GitHub repository secrets before enabling the workflow:

- `SONAR_TOKEN`
- `SONAR_HOST_URL`

The optional quality gate step is included in the workflow and can be removed or commented if the SonarQube/SonarCloud setup does not support waiting for a gate result from GitHub Actions.

---

## 24. Interview Explanation

This project can be explained as:

```text
I built an enterprise asset monitoring platform using microservices.
The system monitors assets, receives telemetry, exposes Prometheus metrics,
generates Prometheus rules dynamically from PostgreSQL, evaluates alerts through
Prometheus, routes alerts through Alertmanager, and persists alert lifecycle in
Alert Service.

I used Go for high-performance services such as asset, telemetry, alert, and rule processing.
Node.js is used for API Gateway and Auth Service.
Python FastAPI is used for reporting.
React is used for the dashboard.
PostgreSQL stores business data.
Prometheus, Alertmanager, and Grafana provide observability and alerting.
JWT and RBAC secure the APIs.
Audit logging provides traceability.

The project is close to a production monitoring architecture because alerting is
metric-driven rather than only application-code-driven.
```

---

## 25. Completed Features

- [x] Docker Compose setup
- [x] Asset Service
- [x] Telemetry Service
- [x] Alert Service
- [x] Rule Service
- [x] Auth Service
- [x] API Gateway
- [x] Report Service
- [x] React Dashboard
- [x] JWT authentication
- [x] RBAC
- [x] Audit logging
- [x] Alert deduplication
- [x] Auto alert resolution
- [x] Prometheus metrics
- [x] Alertmanager webhook flow
- [x] Dynamic Monitoring Rules
- [x] Prometheus dynamic rule generation
- [x] Grafana dashboard support
- [x] Makefile helper commands
- [x] Seed script
- [x] E2E smoke test

---

## 26. Current Limitations

- Dynamic rules currently support numeric telemetry metrics only: `temperature`, `cpu`, and `memory`.
- String rule support such as `status == DOWN` can be added later.
- Dynamic rule generation writes to local mounted YAML files.
- Rule versioning is not yet implemented.
- Dashboard rule management UI is not yet implemented.
- Notification channels such as email, Slack, or Teams are not yet implemented.

---

## 27. Future Improvements

- Kafka or RabbitMQ event-driven communication
- OpenTelemetry distributed tracing
- Centralized logging with ELK or Loki
- Kubernetes deployment
- CI/CD pipeline improvements
- Email, Slack, or Teams notifications
- Asset-specific threshold configuration
- Refresh token support
- String-based rules such as `status == DOWN`
- Rule versioning
- Rule approval workflow
- Dashboard UI for rule management
- Integration tests for dynamic rule generation
- Prometheus rule syntax validation before reload
- Notification Service
- Maintenance windows and alert suppression
