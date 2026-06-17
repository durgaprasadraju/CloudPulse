# ADR-001: Microservices vs Monolith and Database Schema Strategy

- **Status:** Accepted
- **Date:** 2026-06-16
- **Deciders:** CloudPulse platform team

## Context

CloudPulse is an observability platform that ingests metrics/events (collector), analyzes them (analyzer), correlates signals (correlator), exposes a public API (api-gateway), and dispatches alerts (notifier). We must choose an application topology and a data ownership model that supports independent deployment, team scaling, and operational clarity.

## Decision 1: Microservices over monolith

We adopt **domain-aligned microservices** in `/services` rather than a single deployable monolith.

| Option | Pros | Cons |
|--------|------|------|
| **Monolith** | Simple local dev, one deploy unit, easy transactions | Scaling is all-or-nothing; blast radius is large; teams block each other |
| **Microservices (chosen)** | Independent scale/deploy per domain; clearer ownership; fits AWS/K8s roadmap | Network latency, distributed tracing required, more infra |

### Rationale

1. **Different scaling profiles** — collector and pinger workloads are I/O-heavy and bursty; notifier is outbound I/O; api-gateway is request/response. A monolith forces one replica size for all.
2. **Failure isolation** — a bug in Excel import or alert fan-out must not take down ingestion.
3. **Team velocity** — services map to bounded contexts (ingest, analyze, correlate, notify, edge).
4. **12+ year ops maturity** — we accept operational cost and mitigate with Docker Compose (dev), Prometheus/Grafana, and Helm (prod).

Internal communication uses **gRPC**; external clients use **HTTP via api-gateway**.

## Decision 2: Database schema strategy

We use **PostgreSQL as the system of record** with **schema-per-service** in development and **database-per-service** (or schema-per-service with dedicated roles) in production.

| Service | Dev schema | Primary data |
|---------|------------|--------------|
| collector | `collector` | raw check results, ingestion cursors |
| analyzer | `analyzer` | aggregates, SLO rollups |
| correlator | `correlator` | incident graphs, correlation keys |
| api-gateway | `gateway` | API keys, auth metadata (no domain tables) |
| notifier | _(Redis + optional PG audit)_ | delivery outbox, notification audit |

**Redis** is used for queues, pub/sub between collector → analyzer → correlator → notifier, and short-lived caches.

**Repository pattern** — each service defines a `Store` interface with Postgres (GORM) implementation; Mongo remains optional for high-volume telemetry via `DB_TYPE` where justified.

### Rationale

1. **No shared tables** — services do not JOIN across boundaries; they exchange events or call gRPC.
2. **Schema-per-service in dev** — one Postgres container keeps laptop setup simple (`docker compose up`).
3. **Migrations owned by service** — each module ships its own SQL/GORM migrations; no central DBA bottleneck.
4. **Escape hatch** — read replicas and CQRS projections can be added per service without rewriting the monolith.

## Consequences

### Positive

- Clear deploy and ownership boundaries
- Horizontal scale per service on AWS/EKS
- ADR trail for future hires and audits

### Negative

- Local stack requires `docker compose` for full fidelity
- Distributed transactions avoided; we use idempotent consumers and outbox pattern
- More CI pipelines (one per service + web)

## Alternatives considered

- **Modular monolith** — rejected for now; may revisit if team size stays ≤ 2
- **Single shared `public` schema** — rejected; creates hidden coupling
- **Mongo-only** — rejected as default; kept as optional store for telemetry volume

## References

- [docs/ARCHITECTURE.md](./ARCHITECTURE.md)
- [infra/terraform/README.md](../infra/terraform/README.md)
