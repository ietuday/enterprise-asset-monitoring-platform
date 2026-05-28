# v1.4.0 - Notification Service and Alert Delivery

## Overview

This release introduces Notification Service support for alert and incident workflows. Users can now configure email and webhook channels, send test notifications, track notification delivery history, and retry failed notification deliveries.

## Features

- Added notification-service microservice
- Added notification channel management
- Added EMAIL and WEBHOOK notification channels
- Added notification delivery history
- Added notification retry support
- Added test notification action
- Added dashboard Notifications page
- Integrated notifications with CRITICAL alert creation
- Integrated notifications with incident lifecycle events
- Added API Gateway routes for notification APIs

## Validation

- notification-service tests passed
- alert-service tests passed
- dashboard lint passed
- dashboard build passed
- api-gateway syntax check passed
- docker compose build passed

## Notes

- Email delivery requires SMTP environment variables.
- If SMTP is not configured, email notifications fail gracefully and are recorded in notification history.
- Webhook delivery can be tested locally using a simple HTTP test server.
