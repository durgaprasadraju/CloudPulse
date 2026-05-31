# CloudPulse

> A high-performance, cloud-native observability platform built for the modern web.

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![React](https://img.shields.io/badge/React-19+-61DAFB?style=flat&logo=react)](https://react.dev/)

## Overview

CloudPulse is an open-source monitoring engine designed to track website uptime, latency, and system health. It uses **Go's concurrency model** for efficient checks and a **React + TypeScript** dashboard for real-time visibility.

| Service    | URL (Docker `make dev`) |
|------------|-------------------------|
| Frontend   | http://localhost:3000   |
| Backend API | http://localhost:8080  |
| PostgreSQL | `localhost:5432` (`beacon` / `beacon`) |

## Quick start

```bash
git clone https://github.com/yourusername/cloudpulse.git
cd cloudpulse
cp .env.example .env
make dev-build
```

Local development without Docker:

```bash
make backend-run                    # API on :8080
cd frontend && npm install && npm run dev   # UI on :5173
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for full setup and verification steps.

## Repository layout

```
├── backend/     # Go API — cmd/, internal/, pkg/
├── frontend/    # Vite + React + TypeScript + Tailwind
├── deploy/      # Docker Compose & Postgres init
└── Makefile
```

## Tech stack

- **Backend:** Go, [Chi](https://github.com/go-chi/chi), PostgreSQL (planned)
- **Frontend:** React, TypeScript, Tailwind CSS, Vite
- **Deploy:** Docker, Docker Compose

## Roadmap

- [ ] **Phase 1:** Core pinger engine & React dashboard
- [ ] **Phase 2:** Goroutine-based checks & database integration
- [ ] **Phase 3:** AWS infrastructure & Terraform
- [ ] **Phase 4:** Kubernetes deployment & scaling
- [ ] **Phase 5:** AI-assisted root cause analysis & Grafana dashboards

## Community & governance

| Document | Description |
|----------|-------------|
| [CONTRIBUTING.md](CONTRIBUTING.md) | How to contribute code and docs |
| [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) | Community standards |
| [SECURITY.md](SECURITY.md) | Vulnerability reporting |
| [GOVERNANCE.md](GOVERNANCE.md) | Project roles and decisions |
| [AUTHORS.md](AUTHORS.md) | Maintainers and attribution |

## License

Copyright © 2026 **Durga Prasad Raju Nadimpalli**

CloudPulse is licensed under the **Apache License 2.0**. See [LICENSE](LICENSE) and [NOTICE](NOTICE).

We chose Apache 2.0 to align with the cloud-native ecosystem (Kubernetes, Prometheus, Grafana):

- Explicit **patent grant** for contributors and users
- **Enterprise-friendly** terms for production deployments
- Clear expectations for **derivative works** and attribution

Source files include Apache 2.0 license headers. By contributing, you agree your work is licensed under the same terms.

## Maintainer

**Durga Prasad Raju Nadimpalli** — Project lead ([AUTHORS.md](AUTHORS.md))
