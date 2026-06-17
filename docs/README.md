# CloudPulse Monorepo

```
CloudPulse/
├── services/          # Go microservices (go.work)
│   ├── api-gateway/   # HTTP edge, JWT, routing
│   ├── collector/     # Ingestion & URL checks
│   ├── analyzer/      # Metrics analysis & rollups
│   ├── correlator/    # Incident correlation
│   └── notifier/      # Alert delivery
├── web/               # Next.js application (Vercel)
├── infra/             # Terraform, Ansible, Helm, Docker assets
├── docs/              # ADRs, runbooks, architecture
├── platform/          # Shared Go config (CORS, env)
└── packages/          # Shared TypeScript types
```

## Quick start

```bash
# Full stack (Postgres, Redis, Prometheus, Grafana, all services)
docker compose up --build

# Go workspace
go work sync
cd services/api-gateway && go run ./cmd/api-gateway

# Web
cd web && npm install && npm run dev
```

## ADRs

| ID | Title |
|----|-------|
| [Day 1 engineering log](./day1/README.md) | Architecture, components, diagrams, future roadmap |
| [ADR-001](./adr/ADR-001-microservices-and-database-strategy.md) | Microservices vs monolith, DB schema strategy |

## Legacy paths

`backend/`, `frontend/`, and `apps/` remain during migration. New work lands in `services/` and `web/`.
