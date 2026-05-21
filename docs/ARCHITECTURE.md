# Architecture

This document explains the architecture of the Enterprise Asset Monitoring Platform.

---

## 1. High-Level Architecture

```mermaid
flowchart TD
    User[User / Operator] --> Dashboard[React Dashboard]

    Dashboard --> Gateway[Node.js API Gateway]

    Gateway --> Auth[Auth Service - Node.js]
    Gateway --> Asset[Asset Service - Go]
    Gateway --> Telemetry[Telemetry Service - Go]
    Gateway --> Alert[Alert Service - Go]
    Gateway --> Report[Report Service - FastAPI]

    Auth --> DB[(PostgreSQL)]
    Asset --> DB
    Telemetry --> DB
    Alert --> DB
    Report --> DB

    Telemetry --> Alert

    Asset --> Prometheus[Prometheus]
    Telemetry --> Prometheus
    Alert --> Prometheus

    Prometheus --> Grafana[Grafana]
```

---

## 2. Request Flow

```mermaid
sequenceDiagram
    participant U as User
    participant UI as React Dashboard
    participant GW as API Gateway
    participant AUTH as Auth Service
    participant SVC as Backend Service
    participant DB as PostgreSQL

    U->>UI: Login
    UI->>GW: POST /api/auth/login
    GW->>AUTH: POST /auth/login
    AUTH->>DB: Validate user
    DB-->>AUTH: User found
    AUTH-->>GW: JWT token
    GW-->>UI: JWT token

    U->>UI: Access protected page
    UI->>GW: GET /api/assets with Bearer token
    GW->>GW: Validate JWT and RBAC
    GW->>SVC: Forward request
    SVC->>DB: Query data
    DB-->>SVC: Data
    SVC-->>GW: Response
    GW-->>UI: Response
```

---

## 3. Telemetry to Alert Flow

```mermaid
sequenceDiagram
    participant Device as Device / Simulator
    participant GW as API Gateway
    participant TEL as Telemetry Service
    participant ALERT as Alert Service
    participant DB as PostgreSQL

    Device->>GW: POST /api/telemetry
    GW->>GW: Validate JWT and role
    GW->>TEL: Forward telemetry
    TEL->>DB: Store telemetry

    TEL->>TEL: Evaluate rules

    alt Threshold crossed
        TEL->>ALERT: POST /alerts
        ALERT->>DB: Check active duplicate alert
        alt No active duplicate
            ALERT->>DB: Create alert
        else Active alert exists
            ALERT-->>TEL: Return existing alert
        end
    else Telemetry normal
        TEL->>ALERT: PUT /alerts/resolve-active
        ALERT->>DB: Resolve active alert
    end

    TEL-->>GW: Telemetry response
    GW-->>Device: Response
```

---

## 4. Alert Lifecycle

```mermaid
stateDiagram-v2
    [*] --> OPEN
    OPEN --> ACKNOWLEDGED
    ACKNOWLEDGED --> RESOLVED
    OPEN --> RESOLVED
    RESOLVED --> [*]
```

---

## 5. Security Flow

```mermaid
flowchart TD
    Request[Client Request] --> Gateway[API Gateway]
    Gateway --> HasToken{Bearer Token?}

    HasToken -- No --> Unauthorized[401 Unauthorized]
    HasToken -- Yes --> Validate[Validate JWT]

    Validate --> ValidToken{Valid Token?}
    ValidToken -- No --> Unauthorized
    ValidToken -- Yes --> RoleCheck[Check User Role]

    RoleCheck --> Allowed{Allowed Role?}
    Allowed -- No --> Forbidden[403 Forbidden]
    Allowed -- Yes --> Backend[Forward to Backend Service]
```

---

## 6. Observability Flow

```mermaid
flowchart LR
    Asset[Asset Service /metrics] --> Prometheus
    Telemetry[Telemetry Service /metrics] --> Prometheus
    Alert[Alert Service /metrics] --> Prometheus

    Prometheus --> Grafana

    Prometheus --> RuntimeMetrics[Go Runtime Metrics]
    Prometheus --> BusinessMetrics[Business Metrics]

    BusinessMetrics --> TelemetryMetric[telemetry_received_total]
    BusinessMetrics --> AlertCreatedMetric[alerts_created_total]
    BusinessMetrics --> AlertResolvedMetric[alerts_resolved_total]
```

---

## 7. Service Responsibilities

| Service | Responsibility |
|---|---|
| React Dashboard | User interface |
| API Gateway | Routing, JWT validation, RBAC, audit logging |
| Auth Service | Register, login, JWT generation |
| Asset Service | Asset CRUD |
| Telemetry Service | Telemetry ingestion and rule evaluation |
| Alert Service | Alert lifecycle, deduplication, auto-resolution |
| Report Service | Summary and reporting APIs |
| PostgreSQL | Persistent data storage |
| Prometheus | Metrics scraping |
| Grafana | Metrics visualization |

---

## 8. Deployment View

```mermaid
flowchart TD
    subgraph DockerCompose[Docker Compose]
        Dashboard[monitoring-dashboard]
        Gateway[monitoring-api-gateway]
        Auth[monitoring-auth-service]
        Asset[monitoring-asset-service]
        Telemetry[monitoring-telemetry-service]
        Alert[monitoring-alert-service]
        Report[monitoring-report-service]
        Postgres[monitoring-postgres]
        Redis[monitoring-redis]
        Prometheus[monitoring-prometheus]
        Grafana[monitoring-grafana]
    end

    Dashboard --> Gateway
    Gateway --> Auth
    Gateway --> Asset
    Gateway --> Telemetry
    Gateway --> Alert
    Gateway --> Report

    Auth --> Postgres
    Asset --> Postgres
    Telemetry --> Postgres
    Alert --> Postgres
    Report --> Postgres

    Prometheus --> Asset
    Prometheus --> Telemetry
    Prometheus --> Alert
    Grafana --> Prometheus
```

---

## 9. Production Considerations

Future production improvements:

- Kubernetes deployment
- Managed PostgreSQL
- Redis cache usage
- Kafka or RabbitMQ for event-driven communication
- OpenTelemetry distributed tracing
- Centralized logging with Loki or ELK
- Secrets Manager or Vault
- CI/CD pipeline
- Automated integration tests