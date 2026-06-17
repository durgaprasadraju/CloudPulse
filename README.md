# CloudPulse

Cloud-native observability monorepo.

## Layout

| Path | Purpose |
|------|---------|
| `services/` | Go microservices (`go.work`) |
| `web/` | Next.js app (Vercel) |
| `infra/` | Terraform, Ansible, Helm, Docker |
| `docs/` | ADRs, commit conventions, GitHub setup |

## Services

| Service | Port | Role |
|---------|------|------|
| api-gateway | 8080 | HTTP edge, JWT, routing |
| collector | 8081 | Ingestion & checks |
| analyzer | 8082 | Metrics analysis |
| correlator | 8083 | Incident correlation |
| notifier | 8084 | Alerts |

## Quick start

```bash
docker compose up --build
cd web && npm install && npm run dev    # http://localhost:3000
```

Grafana: http://localhost:3001 (admin/admin) · Prometheus: http://localhost:9090

## Docs

- [Day 1 engineering log](docs/day1/README.md) — architecture, components, diagrams
- [Monorepo index](docs/README.md)
- [ADR-001: Microservices & DB strategy](docs/adr/ADR-001-microservices-and-database-strategy.md)
- [Conventional commits](docs/COMMITS.md)
- [GitHub push & branch protection](docs/GITHUB_SETUP.md)
