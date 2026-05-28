# v1.5.0 - SLA and Escalation Engine

## Overview

This release adds SLA tracking and escalation workflows for incidents. Incidents now receive acknowledgement and resolution deadlines based on severity. The platform can automatically detect SLA breaches, record escalation history, and send escalation notifications through the notification-service.

## Features

- Added SLA policy management
- Added incident SLA tracking
- Added acknowledgement and resolution deadlines
- Added SLA status tracking
- Added automatic SLA breach detection
- Added escalation history
- Added manual incident escalation
- Added notification events for SLA breaches and escalations
- Added API Gateway routes for SLA APIs
- Added dashboard SLA page

## Validation

- alert-service tests passed
- api-gateway syntax check passed
- dashboard lint passed
- dashboard build passed
- docker compose build passed

## Notes

- SLA worker interval is configurable using SLA_CHECK_INTERVAL_SECONDS.
- SLA worker can be disabled using SLA_WORKER_ENABLED=false.
- Notification failures do not block incident or escalation workflows.
