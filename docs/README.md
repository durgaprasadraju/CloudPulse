# CloudPulse Documentation

Ride-sharing platform for **CloudPulse.live** — Go microservices, Next.js frontend, RabbitMQ event bus, MongoDB (trips), PostgreSQL (driver registry), Redis (driver locations), Docker Compose (local), Terraform + Helm + ArgoCD (AWS).

## Index

### Architecture
| Document | Description |
|----------|-------------|
| [System overview](./architecture/system-overview.md) | C4-style design, components, ports, tech choices |
| [Data & event flow](./architecture/data-flow.md) | Exact exchanges, queues, routing keys, payloads |
| [Trip creation flow v1](./architecture/trip-creation-flow-v1.md) | Sequence diagram (legacy; see data-flow for code-accurate keys) |
| [RabbitMQ flow v1](./architecture/rabbitmq-flow-v1.md) | Historical topology sketch |

### Services (one file each)
| Document | Service |
|----------|---------|
| [API Gateway](./services/api-gateway.md) | HTTP + WebSocket edge |
| [Trip Service](./services/trip-service.md) | Routing, fares, trip lifecycle |
| [Driver Service](./services/driver-service.md) | Matching & driver registry |
| [Payment Service](./services/payment-service.md) | Checkout session creation |
| [Web (Frontend)](./services/web.md) | Rider / Driver UI |

### Simulation & testing
| Document | Description |
|----------|-------------|
| [Local E2E simulation](./simulation/local-e2e-test.md) | Step-by-step test of the full ride flow |
| [Failure & edge cases](./simulation/edge-cases.md) | No drivers, decline, payment mock, reconnect |

### Ops (elsewhere in repo)
- Local stack: root `docker-compose.yaml`, `.env.example`
- AWS infra: `terraform/`
- K8s deploy: `charts/cloudpulse/`, `gitops/`
- CI/CD: `.github/workflows/ci-cd.yaml`, `gitops/README.md`
