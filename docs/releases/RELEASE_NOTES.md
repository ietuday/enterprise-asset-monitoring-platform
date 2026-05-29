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
