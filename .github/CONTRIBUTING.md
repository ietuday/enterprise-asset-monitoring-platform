# Contributing

Thank you for contributing to the Enterprise Asset Monitoring Platform.

This guide explains the development workflow, coding standards, local checks, and pull request process for this project.

---

## Development Flow

1. Create a feature branch from `master`.

```bash
git checkout master
git pull origin master
git checkout -b feat/your-feature-name
````

2. Make your changes.

3. Run local validation before committing.

```bash
make config
make test
make test-api
make test-metrics
cd web/dashboard && npm run build
cd ../..
```

4. Commit changes using conventional commit style.

```bash
git add .
git commit -m "feat: add your feature"
```

5. Push your branch.

```bash
git push origin feat/your-feature-name
```

6. Open a pull request into `master`.

7. Wait for CI to pass.

8. Request review and merge after approval.

---

## Branch Naming

Use clear branch names:

```text
feat/dynamic-rules-ui
fix/alertmanager-webhook
test/e2e-dynamic-rules
docs/update-readme
ci/github-actions
chore/cleanup
```

Recommended prefixes:

| Prefix      | Purpose                |
| ----------- | ---------------------- |
| `feat/`     | New feature            |
| `fix/`      | Bug fix                |
| `test/`     | Test changes           |
| `docs/`     | Documentation          |
| `ci/`       | CI/CD changes          |
| `chore/`    | Cleanup or maintenance |
| `refactor/` | Code refactoring       |

---

## Commit Message Style

Use conventional commit style:

```text
feat: add dynamic monitoring rules
fix: resolve alertmanager webhook parsing
test: add e2e smoke test for dynamic rules
docs: update readme with ci instructions
ci: optimize github actions workflow
chore: format go services
refactor: simplify telemetry handler
```

---

## Local Setup

Start all services:

```bash
docker compose up -d --build
```

Check running containers:

```bash
docker ps
```

Stop services:

```bash
docker compose down
```

Stop services and remove volumes:

```bash
docker compose down -v
```

---

## Environment Configuration

Copy the example environment file:

```bash
cp .env.example .env
```

Do not commit `.env`.

Use `.env.example` for safe sample values only.

---

## Local Checks

Before opening a PR, run:

```bash
make config
make test
make test-api
make test-metrics
```

For dashboard changes:

```bash
cd web/dashboard
npm run build
```

For Docker changes:

```bash
docker compose config
docker compose build
```

For shell script changes:

```bash
bash -n scripts/e2e-smoke-test.sh
bash -n scripts/seed.sh
```

---

## Go Services

Go services:

```text
services/asset-service
services/telemetry-service
services/alert-service
services/rule-service
```

Format Go code:

```bash
gofmt -w services/asset-service \
          services/telemetry-service \
          services/alert-service \
          services/rule-service
```

Run Go tests:

```bash
make test
```

Or run for one service:

```bash
cd services/rule-service
go test ./...
```

Build one Go service:

```bash
go build -o /tmp/rule-service ./cmd
```

---

## Node.js Services

Node.js services:

```text
services/api-gateway
services/auth-service
```

Install dependencies:

```bash
npm ci
```

Run lint if available:

```bash
npm run lint --if-present
```

Run tests if available:

```bash
npm test --if-present
```

---

## React Dashboard

Dashboard path:

```text
web/dashboard
```

Install dependencies:

```bash
cd web/dashboard
npm ci
```

Run build:

```bash
npm run build
```

---

## Python Report Service

Report service path:

```text
services/report-service
```

Install dependencies:

```bash
cd services/report-service
pip install -r requirements.txt
```

Run import check:

```bash
python -m py_compile main.py
```

---

## Testing

Run all Go service tests:

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

Run E2E smoke test:

```bash
make e2e
```

The E2E smoke test validates:

* Admin login
* Asset creation
* Telemetry ingestion
* Prometheus metric update
* Alertmanager-based alert creation
* Alert auto-resolution
* Dynamic rule creation
* Rule audit history

---

## Dynamic Monitoring Rules

Dynamic Monitoring Rules allow monitoring thresholds to be managed from APIs instead of hardcoded logic.

Supported rule metrics:

```text
temperature
cpu
memory
status
```

Numeric rules use `threshold`.

Example:

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

Status rules use `value`.

Example:

```json
{
  "name": "Dynamic Device Down",
  "metric": "status",
  "operator": "==",
  "value": "DOWN",
  "severity": "CRITICAL",
  "enabled": true
}
```

For details, see:

```text
docs/DYNAMIC-RULES.md
```

---

## CI/CD

GitHub Actions runs on:

```text
push to main, master, develop
pull request to main, master, develop
```

CI validates:

* Docker Compose config
* Shell script syntax
* Go formatting
* Go tests
* Go builds
* Node dependency install
* Node lint/tests if available
* Python import check
* React dashboard build
* Docker image build

Before pushing, make sure the same checks pass locally.

---

## Pull Request Checklist

Before opening a pull request:

* [ ] Code is formatted
* [ ] Tests pass locally
* [ ] Docker Compose config is valid
* [ ] Dashboard builds successfully if frontend changed
* [ ] README/docs updated if needed
* [ ] No secrets committed
* [ ] CI is green
* [ ] Screenshots added for UI changes
* [ ] E2E test updated if behavior changed

---

## Secrets Policy

Never commit:

* `.env`
* JWT secrets
* database passwords
* API keys
* GitHub tokens
* private keys
* credentials

Use `.env.example` for safe example values.

---

## Protected Branch Rules

The `master` branch is protected.

Expected workflow:

```text
feature branch -> pull request -> CI checks -> approval -> merge
```

Direct pushes to `master` should be avoided.

---

## Useful Commands

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
make e2e
make seed
make db-shell
make clean
```

---

## Need Help?

Check these files first:

```text
README.md
docs/DYNAMIC-RULES.md
docs/API-TESTING.md
docs/ARCHITECTURE.md
SECURITY.md
```

````

Then commit:

```bash
git add CONTRIBUTING.md
git commit -m "docs: add contributing guide"
git push origin master
````
