# Security policy

## Supported versions

Security fixes are applied to the latest release on the `main` branch. Older tags may
receive fixes at maintainer discretion depending on severity and adoption.

| Version | Supported          |
| ------- | ------------------ |
| main    | :white_check_mark: |
| < 1.0   | :x: (pre-release)  |

## Reporting a vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

If you discover a security issue in CloudPulse, report it responsibly so we can
investigate and release a fix before details are public.

### How to report

1. **Preferred:** Open a private security advisory on GitHub  
   `Security` → `Advisories` → `Report a vulnerability`  
   (when the repository is published on GitHub)

2. **Alternative:** Contact the project lead directly:  
   **Durga Prasad Raju Nadimpalli**  
   Include:
   - Description of the vulnerability
   - Steps to reproduce
   - Impact assessment (if known)
   - Affected versions or components (backend, frontend, deploy)

### What to expect

- **Acknowledgment** within 3 business days
- **Initial assessment** within 7 business days
- **Coordinated disclosure** — we will work with you on timing and credit (if desired)
- **Fix and advisory** for confirmed issues, published after a patch is available

## Security best practices for deployments

When self-hosting CloudPulse:

- Change default database credentials in `deploy/docker-compose.yml`
- Do not expose PostgreSQL publicly without TLS and network restrictions
- Run services behind a reverse proxy with TLS termination
- Keep Docker images and dependencies updated
- Store secrets in environment variables or a secrets manager, not in Git

## Scope

The following are generally **in scope**:

- Authentication or authorization flaws in CloudPulse services
- Injection, SSRF, or RCE in application code
- Sensitive data exposure via APIs or configuration

The following are generally **out of scope**:

- Vulnerabilities in third-party dependencies already fixed in a newer upstream release
  (please report upstream; we will bump versions)
- Issues requiring physical access to a host
- Social engineering attacks

Thank you for helping keep CloudPulse and its users safe.
