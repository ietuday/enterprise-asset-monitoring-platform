# v1.6.0 - Preventive Maintenance and Asset Health

## Overview

This release adds preventive maintenance and asset health scoring. The platform can now schedule and track work before failures occur, then reflect overdue maintenance and operational risk in report-service health scores.

## Added

- New maintenance-service for preventive maintenance tasks
- Maintenance task lifecycle: scheduled, in progress, completed, overdue, cancelled
- Maintenance history tracking
- API Gateway routing and RBAC for maintenance APIs
- Maintenance audit action mapping
- Maintenance dashboard page
- Asset Health Score endpoint in report-service
- API smoke coverage for maintenance create, complete, and history
- Playwright UI coverage for maintenance task creation and completion

## APIs Added

- `GET /maintenance/tasks`
- `POST /maintenance/tasks`
- `GET /maintenance/tasks/{id}`
- `PUT /maintenance/tasks/{id}`
- `PATCH /maintenance/tasks/{id}/status`
- `POST /maintenance/tasks/{id}/complete`
- `POST /maintenance/tasks/{id}/cancel`
- `GET /maintenance/assets/{assetId}/tasks`
- `GET /maintenance/overdue`
- `GET /maintenance/history/{taskId}`
- `GET /reports/asset-health`
- `GET /reports/asset-health/{asset_id}`

## Database Tables Added

- `maintenance_tasks`
- `maintenance_history`

## Notes

- Maintenance tasks store `asset_id` as text to match the existing asset-service schema.
- Asset Health Score is a simple rules-based score; ML, forecasting, calendars, and work-order integrations are intentionally out of scope for this release.
