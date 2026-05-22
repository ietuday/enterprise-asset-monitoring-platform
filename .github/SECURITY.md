# Security Policy

## Supported Branch

Security fixes are maintained on the following branch:

```text
master
````

---

## Reporting a Vulnerability

Please do not open a public GitHub issue for sensitive security problems.

For this learning/demo project, report security concerns directly to the repository owner.

If you discover a vulnerability, please include:

* A clear description of the issue
* Steps to reproduce
* Affected service or component
* Possible impact
* Suggested fix, if available

---

## Security Practices

This project uses the following security practices:

* JWT authentication
* Role-based access control
* Protected branch rules
* Required CI checks before merging
* Dependency update automation through Dependabot
* Dockerized services
* Audit logging for API requests
* Rule audit history for Dynamic Monitoring Rules
* Environment-based configuration

---

## Secrets Policy

Do not commit secrets such as:

* `.env` files
* JWT secrets
* Database passwords
* API keys
* GitHub tokens
* Private keys
* Cloud credentials

Use `.env.example` for safe sample configuration values.

---

## Local Environment

Create a local environment file from the example file:

```bash
cp .env.example .env
```

The `.env` file should remain local and should not be committed.

---

## Dependency Updates

Dependencies are monitored through Dependabot.

Dependabot may create pull requests for:

* Go modules
* Node.js packages
* Python packages
* Docker images
* GitHub Actions

All dependency update pull requests should pass CI before merging.

---

## Branch Protection

The `master` branch is protected.

Expected workflow:

```text
feature branch -> pull request -> CI checks -> approval -> merge
```

Direct pushes to `master` should be avoided.

---

## CI Security Checks

The CI workflow validates:

* Docker Compose configuration
* Shell script syntax
* Go formatting
* Go tests
* Go builds
* Node dependency installation
* Node lint/tests if available
* Python import check
* React dashboard build
* Docker image build

---

## Known Limitations

This is a learning/demo project. It does not yet include:

* Production-grade secret management
* OAuth/OIDC authentication
* Refresh token rotation
* Rate limiting
* WAF integration
* CodeQL scanning
* Container image vulnerability scanning
* Centralized security monitoring

These can be added as future improvements.

---

## Future Security Improvements

Planned improvements:

* Add CodeQL workflow
* Add container vulnerability scanning
* Add API rate limiting
* Add refresh token support
* Add password hashing policy documentation
* Add secret scanning
* Add production-ready secrets management
* Add HTTPS/TLS deployment guide
* Add audit log retention policy

````

Then commit:

```bash
git add SECURITY.md
git commit -m "docs: add security policy"
git push origin master
````
