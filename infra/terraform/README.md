# CloudPulse Terraform (AWS stub)

Terraform modules for production will live here:

- `modules/vpc` — networking
- `modules/eks` — Kubernetes cluster
- `modules/rds` — Postgres per-service databases
- `modules/elasticache` — Redis

## Usage (future)

```bash
cd infra/terraform/environments/dev
terraform init
terraform plan
```
