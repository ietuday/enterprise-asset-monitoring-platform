# v1.7.0 - Predictive Maintenance Insights

## Overview

This release introduces Predictive Maintenance Insights, helping users identify risky assets using asset health scores, maintenance task status, and maintenance history.

## New Features

- Added Maintenance Insights API in report-service.
- Added deterministic risk scoring: low, medium, high, and critical.
- Added overdue maintenance risk escalation.
- Added recommended maintenance actions for each asset.
- Added Dashboard Maintenance Insights table.
- Added API Gateway route for maintenance insights.

## API

New endpoint:

```text
GET /api/reports/maintenance-insights
```

Example response:

```json
[
  {
    "asset_id": "1",
    "asset_name": "Pump A",
    "health_score": 45,
    "risk_level": "high",
    "last_maintenance_date": "2026-05-20",
    "open_tasks": 2,
    "overdue_tasks": 1,
    "recommended_action": "Schedule preventive maintenance within 7 days",
    "reason": "Asset health score is low at 45 and there are 1 overdue maintenance task"
  }
]
```

## Security and Governance

- Added RBAC mapping for maintenance insights.
- Added audit mapping for maintenance insight access.

## Testing

- Added report-service unit tests for risk calculation and recommendations.
- Added report-service API tests.
- Added API Gateway route tests.
- Added React dashboard tests.
- Added Playwright coverage for Maintenance Insights.
- Added API smoke coverage.

## Compatibility

- Existing v1.6.0 preventive maintenance and asset health functionality remains unchanged.

---

# v1.6.0 - Preventive Maintenance and Asset Health

## Added

- New maintenance-service for preventive maintenance tasks.
- Maintenance task lifecycle: scheduled, in progress, completed, overdue, cancelled.
- Maintenance history tracking.
- API Gateway routing and RBAC for maintenance APIs.
- Maintenance dashboard page.
- Asset Health Score endpoint in report-service.
- E2E smoke/UI coverage for maintenance workflow.

---

# v1.1.0 - CI, Security, and Repository Governance

## Highlights

This release improves the project’s production-readiness by adding stronger CI/CD, security scanning, repository governance, and contributor documentation.

## Added

- GitHub Actions CI workflow
- Go service checks for:
  - asset-service
  - telemetry-service
  - alert-service
  - rule-service
- Node.js service checks for:
  - api-gateway
  - auth-service
- Python report-service validation
- React dashboard build validation
- Docker Compose validation and Docker image build
- Trivy security scanning workflow
- Gitleaks secret scanning workflow
- Dependabot configuration
- Branch ruleset setup script
- Pull request template
- Bug report template
- Feature request template
- CODEOWNERS
- CONTRIBUTING.md
- SECURITY.md
- CI badge support in README

## Improved

- GitHub Actions workflow caching
- Go build output path handling
- Go formatting checks
- Shell script syntax validation
- Security workflow permissions
- Repository branch protection setup
- Documentation for contribution, security, and local validation

## Fixed

- Go CI build issue where output binary conflicted with `cmd` directory
- Formatting issues in Go service files
- Workflow newline and hidden-character warnings
- GitHub Advanced Security workflow permission warnings

## Notes

The repository now has a stronger development workflow:

```text
feature branch -> pull request -> CI checks -> approval -> merge
