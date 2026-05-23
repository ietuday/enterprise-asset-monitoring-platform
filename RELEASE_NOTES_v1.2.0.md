# v1.2.0 - Rule Lifecycle Management

## Overview

This release introduces lifecycle management for dynamic monitoring rules in the Enterprise Asset Monitoring Platform.

Rules can now move through clear operational states such as draft, active, disabled, and archived. This improves rule governance, prevents incomplete rules from being pushed to Prometheus, and provides better auditability for rule changes.

## Features

- Added rule lifecycle status:
  - draft
  - active
  - disabled
  - archived

- Added lifecycle APIs:
  - PATCH /rules/{id}/activate
  - PATCH /rules/{id}/disable
  - PATCH /rules/{id}/archive

- Added status-based rule filtering:
  - GET /rules?status=draft
  - GET /rules?status=active
  - GET /rules?status=disabled
  - GET /rules?status=archived

- Updated Prometheus dynamic rule generation:
  - Only active rules are included in generated Prometheus alert rules.
  - Draft, disabled, and archived rules are excluded.

- Added audit events for lifecycle operations:
  - RULE_CREATED
  - RULE_UPDATED
  - RULE_ACTIVATED
  - RULE_DISABLED
  - RULE_ARCHIVED
  - RULE_DELETED

## Behavior Changes

- New rules are created with status `draft` by default.
- A rule becomes enabled only when its status is `active`.
- Disabled and archived rules are not included in Prometheus rule generation.
- Archived rules cannot be updated or reactivated.
- Existing enabled rules are migrated to `active`.
- Existing non-enabled rules are migrated to `draft`.

## Benefits

- Prevents incomplete or invalid rules from triggering alerts.
- Improves operational safety.
- Adds better traceability for rule lifecycle changes.
- Makes rule management more enterprise-ready.


# v1.2.0 - Rule Lifecycle Management, Versioning, and Rollback

## Overview

This release improves dynamic monitoring rule governance by adding lifecycle management, version history, and rollback support.

Rules can now move through clear operational states and every update can be tracked through version snapshots. This makes rule management safer, more auditable, and more enterprise-ready.

## Features

### Rule Lifecycle Management

- Added rule lifecycle statuses:
  - draft
  - active
  - disabled
  - archived

- Added lifecycle APIs:
  - PATCH /rules/{id}/activate
  - PATCH /rules/{id}/disable
  - PATCH /rules/{id}/archive

- Added status-based filtering:
  - GET /rules?status=draft
  - GET /rules?status=active
  - GET /rules?status=disabled
  - GET /rules?status=archived

### Prometheus Rule Generation

- Prometheus dynamic rules are now generated only from active rules.
- Draft, disabled, and archived rules are excluded from Prometheus alert generation.

### Rule Audit History

Added lifecycle audit events:

- RULE_CREATED
- RULE_UPDATED
- RULE_ACTIVATED
- RULE_DISABLED
- RULE_ARCHIVED
- RULE_DELETED
- RULE_ROLLED_BACK

### Rule Versioning and Rollback

- Added rule_versions table
- Added RuleVersion model
- Added version snapshot before rule updates
- Added API to list rule versions:
  - GET /rules/{id}/versions
- Added rollback API:
  - POST /rules/{id}/rollback/{version}

## Behavior Changes

- New rules are created as draft by default.
- A rule becomes enabled only when status is active.
- Archived rules cannot be updated or rolled back.
- Rollback restores previous rule configuration.
- Rollback regenerates Prometheus dynamic rules.

## Testing

Validated with:

- go test ./...
- Manual curl testing for:
  - create rule
  - activate rule
  - disable rule
  - archive rule
  - update rule
  - list rule versions
  - rollback rule
  - audit history