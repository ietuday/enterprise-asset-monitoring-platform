# v1.5.0 - SLA and Escalation Engine

## Overview

This release adds SLA tracking and escalation workflows for incident management. Incidents now receive acknowledgement and resolution deadlines based on severity. The platform can automatically detect SLA breaches, record escalation history, and send escalation notifications through the notification-service.

## Features

- Added SLA policy management
- Added incident SLA tracking
- Added acknowledgement and resolution deadlines
- Added SLA status tracking
- Added automatic SLA breach detection worker
- Added escalation history
- Added manual incident escalation
- Added notification events for SLA breaches and escalations
- Added API Gateway routes for SLA APIs
- Added dashboard SLA page
- Added default SLA policies to seed flow

## APIs Added

- `POST /sla-policies`
- `GET /sla-policies`
- `GET /sla-policies/{id}`
- `PUT /sla-policies/{id}`
- `DELETE /sla-policies/{id}`
- `GET /sla-breaches`
- `GET /incidents/{id}/sla`
- `POST /incidents/{id}/escalate`
- `GET /incidents/{id}/escalations`

## Database Tables Added

- `sla_policies`
- `incident_sla_tracking`
- `escalation_history`

## Validation

- `go fmt ./...` passed in alert-service
- `go test ./...` passed in alert-service
- `go test ./...` passed in notification-service
- `node --check server.js` passed in api-gateway
- `npm run lint -- --quiet` passed in dashboard
- `npm run build` passed in dashboard
- `docker compose build` passed
- `docker compose up -d` passed
- Health checks passed for alert-service and api-gateway

## Notes

- SLA worker interval is configurable using `SLA_CHECK_INTERVAL_SECONDS`.
- SLA worker can be disabled using `SLA_WORKER_ENABLED=false`.
- Notification failures do not block incident or escalation workflows.
- Business calendars, holidays, on-call rotations, and multi-level escalation chains are intentionally out of scope for this release.