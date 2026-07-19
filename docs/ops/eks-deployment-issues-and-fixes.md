# EKS deployment — issues and fixes

Chronological record of problems encountered while taking CloudPulse from local Docker Compose to production on AWS EKS (GitOps with Argo CD), and how each was fixed.

**Environment:** `cloudpulse-production` (EKS 1.30, us-east-1)  
**Domains:** `cloudpulse.live`, `www.cloudpulse.live`, `api.cloudpulse.live`  
**Date range:** July 2026

---

## Table of contents

1. [Local: no drivers assigned after ride select](#1-local-no-drivers-assigned-after-ride-select)
2. [CI: Node 20 deprecation on GitHub Actions](#2-ci-node-20-deprecation-on-github-actions)
3. [CI: Docker Hub login failed](#3-ci-docker-hub-login-failed)
4. [Terraform: `for_each` unknown values (RDS / ElastiCache SG rules)](#4-terraform-for_each-unknown-values-rds--elasticache-sg-rules)
5. [Terraform: ElastiCache transient `InvalidCredentialsException`](#5-terraform-elasticache-transient-invalidcredentialsexception)
6. [Terraform: stale state lock](#6-terraform-stale-state-lock)
7. [DNS / ACM: certificate stuck in `PENDING_VALIDATION`](#7-dns--acm-certificate-stuck-in-pending_validation)
8. [EKS: infrastructure up, app not deployed](#8-eks-infrastructure-up-app-not-deployed)
9. [Terraform language server redlines (`set` blocks / stale index)](#9-terraform-language-server-redlines-set-blocks--stale-index)
10. [Argo CD bootstrap: `AppProject` CRD not found at plan time](#10-argo-cd-bootstrap-appproject-crd-not-found-at-plan-time)
11. [Helm: ALB controller name already in use](#11-helm-alb-controller-name-already-in-use)
12. [Argo CD: `Chart.yaml file is missing`](#12-argo-cd-chartyaml-file-is-missing)
13. [Pods: `ImagePullBackOff` for trip / driver / payment](#13-pods-imagepullbackoff-for-trip--driver--payment)
14. [Domain not resolving: no Route 53 aliases to ALB](#14-domain-not-resolving-no-route-53-aliases-to-alb)
15. [HTTPS 504: ALB targets unhealthy (security groups)](#15-https-504-alb-targets-unhealthy-security-groups)
16. [API target unhealthy: no `/health` route](#16-api-target-unhealthy-no-health-route)
17. [Map WebSocket errors](#17-map-websocket-errors)

---

## 1. Local: no drivers assigned after ride select

### Issue
Selecting a ride in the local UI never progressed to payment. No drivers were assigned.

### Root cause
Driver registry and location tracking were in-memory / unused. Terraform had provisioned RDS and ElastiCache, but the Go services were not wired to them for local or prod-like flows.

### Fix
- Added PostgreSQL persistence in `driver-service` (`postgres.go`, `DATABASE_URL`).
- Added Redis GEO location store in `shared/tracking` (`REDIS_URL`).
- Wired `driver-service` and `api-gateway` WebSocket location updates to Redis.
- Updated `docker-compose.yaml` and `.env.example` so local runs use Postgres + Redis.

---

## 2. CI: Node 20 deprecation on GitHub Actions

### Issue
Workflow warnings: Node 20 is deprecated; runners default to Node 24.

### Fix
Bumped Actions in `.github/workflows/ci-cd.yaml`:

| Action | From | To |
|--------|------|-----|
| `actions/checkout` | v4 | v5 |
| `dorny/paths-filter` | v3 | v4 |
| `docker/setup-buildx-action` | v3 | v4 |
| `docker/login-action` | v3 | v4 |
| `docker/build-push-action` | v6 | v7 |

---

## 3. CI: Docker Hub login failed

### Issue
```
Error: Username and password required
```
from `docker/login-action`.

### Root cause
Missing GitHub repository secrets.

### Fix
Set in the repo settings (Actions secrets):

- `DOCKERHUB_USERNAME`
- `DOCKERHUB_TOKEN` (access token, not account password)

Re-run the workflow after secrets are present.

---

## 4. Terraform: `for_each` unknown values (RDS / ElastiCache SG rules)

### Issue
```
Invalid for_each argument ... depends on resource attributes that cannot be
determined until apply
```
on RDS and ElastiCache security group ingress rules.

### Root cause
`for_each` was driven by a **list** of security group IDs that are unknown until apply. Terraform requires **known keys** for `for_each` at plan time.

### Fix
Changed `allowed_security_group_ids` from `list(string)` to `map(string)` with static keys:

```hcl
allowed_security_group_ids = {
  eks_nodes           = module.eks.node_security_group_id
  eks_cluster_primary = module.eks.cluster_primary_security_group_id
}
```

Updated `modules/rds` and `modules/elasticache` accordingly.

---

## 5. Terraform: ElastiCache transient `InvalidCredentialsException`

### Issue
```
CreateReplicationGroup ... StatusCode: 408 ... InvalidCredentialsException
```

### Root cause
Transient AWS API error (not a bad IAM credential in this case).

### Fix
Re-run `terraform apply`. No code change required.

---

## 6. Terraform: stale state lock

### Issue
```
Error acquiring the state lock
```

### Root cause
A previous `terraform plan` process was still holding the local lock file.

### Fix
Kill the stuck process and remove the stale lock file (`.terraform.tfstate.lock.info` for local state). Then re-run plan/apply.

---

## 7. DNS / ACM: certificate stuck in `PENDING_VALIDATION`

### Issue
ACM certificate for `cloudpulse.live` / SANs stayed `PENDING_VALIDATION` for a long time. `api.cloudpulse.live` lagged behind apex/www.

### Root cause
Registrar nameservers were **not** delegated to the Route 53 hosted zone Terraform created. ACM DNS validation CNAMEs existed in Route 53 but the public internet was not querying that zone.

### Fix
1. Read nameservers: `terraform output route53_name_servers`
2. Update the domain registrar to use those NS records.
3. Wait for ACM to poll again (validation can take additional time per SAN).

Route 53 + ACM were added via `terraform/modules/dns`.

---

## 8. EKS: infrastructure up, app not deployed

### Issue
Cluster and nodes were Ready, but trip/driver/payment/frontend pods were missing. No way to reach the app via the domain.

### Root cause
Terraform had provisioned VPC, EKS, RDS, Redis, DNS — but **Helm add-ons and the app** (ALB Controller, Argo CD, CloudPulse Application) were not installed yet.

### Fix
Added `terraform/modules/eks-addons`:

- AWS Load Balancer Controller (IRSA + Helm)
- Argo CD (Helm)
- Bootstrap of AppProject + Application from `gitops/argocd/`

Enabled flags in `terraform.tfvars` and ran `terraform apply`.

---

## 9. Terraform language server redlines (`set` blocks / stale index)

### Issue
Editor showed errors on `helm_release` `set { }` blocks and phantom undeclared variables in `main.tf` / `tfvars`, while `terraform validate` passed.

### Root cause
IDE Terraform LS was using a newer Helm provider schema (3.x) where `set` is not a block; locked provider was 2.x. Also stale language-server index after new modules.

### Fix
- Rewrote Helm values as `values = [yamlencode({ ... })]` (valid on Helm provider 2.x and 3.x).
- Confirmed with `terraform validate`.
- Reload window / restart Terraform language server if redlines linger.

---

## 10. Argo CD bootstrap: `AppProject` CRD not found at plan time

### Issue
```
API did not recognize GroupVersionKind from manifest (CRD may not be installed)
... no matches for kind "AppProject" in group "argoproj.io"
```
during Terraform plan/apply.

### Root cause
`kubernetes_manifest` validates CRDs **at plan time**. Argo CD CRDs only exist **after** the Helm chart installs.

### Fix
Replaced `kubernetes_manifest` with `null_resource` + `local-exec` that:

1. Waits for CRDs (`kubectl wait ... Established`)
2. Runs `kubectl apply` on `application-project.yaml` and `application.yaml`

---

## 11. Helm: ALB controller name already in use

### Issue
```
Error: cannot re-use a name that is still in use
```
on `helm_release.aws_load_balancer_controller`.

### Root cause
ALB Controller was already installed on the cluster (manual/earlier Helm) and was **not** in Terraform state. Terraform tried to create a second release with the same name.

### Fix
Set in `terraform.tfvars`:

```hcl
enable_aws_load_balancer_controller = false  # already installed
enable_metrics_server               = false  # already installed
```

Left the existing healthy controller in place. Argo CD install/bootstrap still managed by Terraform.

---

## 12. Argo CD: `Chart.yaml file is missing`

### Issue
Argo CD Application sync failed:

```
Error: Chart.yaml file is missing
```

even though `charts/cloudpulse/Chart.yaml` existed in Git.

### Root cause
`.helmignore` contained:

```
!.helmignore
```

That negation pattern breaks Helm’s ignore loader so it never loads `Chart.yaml`. Confirmed locally: `helm template` failed with the same error until the line was removed.

### Fix
- Removed `!.helmignore` from `charts/cloudpulse/.helmignore`.
- Simplified Argo CD Application to a **single source** with a relative values file:

  `../../gitops/cloudpulse/values-production.yaml`

---

## 13. Pods: `ImagePullBackOff` for trip / driver / payment

### Issue
`trip-service`, `driver-service`, and `payment-service` stuck in `ImagePullBackOff`. Frontend/backend (Docker Hub) were fine.

### Root cause
GitOps values pointed at ECR images (`:latest`) that had never been built/pushed. CI only built `api-gateway` and `web`.

### Fix
Built and pushed to ECR:

```text
cloudpulse-trip-service:34072bac8d49 (+ latest)
cloudpulse-driver-service:34072bac8d49 (+ latest)
cloudpulse-payment-service:34072bac8d49 (+ latest)
```

Pinned those tags in `gitops/cloudpulse/values-production.yaml` and let Argo sync / restarted deployments.

---

## 14. Domain not resolving: no Route 53 aliases to ALB

### Issue
`cloudpulse.live` / `api.cloudpulse.live` did not resolve (empty `dig` A records). ACM and NS delegation were already OK.

### Root cause
Route 53 hosted zone only had ACM validation CNAMEs. No A/ALIAS records pointed at the ALB. Terraform `alb_dns_name` / `alb_zone_id` were still empty.

### Fix
After Ingress created the ALB:

```hcl
alb_dns_name = "k8s-cloudpulse-....elb.amazonaws.com"
alb_zone_id  = "Z35SXDOTRQ7X7K"  # ALB canonical zone for us-east-1
```

`terraform apply` created apex, `api`, and `www` alias records.

---

## 15. HTTPS 504: ALB targets unhealthy (security groups)

### Issue
DNS worked; HTTPS returned **504**. Target group health: `Target.Timeout`.

### Root cause
Node security group (`cloudpulse-production-node-sg`) allowed traffic from the cluster SG only. ALB’s shared backend SG could not reach pod ports (3000 frontend, 8080 backend).

### Fix
Authorized ingress on the node SG from the ALB backend SG:

- TCP **3000** (Next.js)
- TCP **8080** (api-gateway)

Targets became healthy; `https://cloudpulse.live` returned **200**.

> **Follow-up (recommended):** encode this as a Terraform security group rule so it is not lost on recreate.

---

## 16. API target unhealthy: no `/health` route

### Issue
Frontend TG healthy; backend TG stayed unhealthy / API returned odd status. ALB health checks hit `/` by default.

### Root cause
`api-gateway` only registered trip/WS/webhook routes — **no GET `/`**. Health checks got non-success responses.

### Fix
- Added `GET /health` → `200 ok` in `services/api-gateway/main.go`.
- Annotated backend Service with  
  `alb.ingress.kubernetes.io/healthcheck-path: /health`.
- Rebuilt/pushed api-gateway image and rolled out via GitOps.

---

## 17. Map WebSocket errors

### Issue
Map UI showed “WebSocket error” while browsing `cloudpulse.live`.

### Investigation
- Frontend bundle already baked in `wss://api.cloudpulse.live/ws` (correct).
- Manual upgrade with HTTP/1.1 returned **101** for `/ws/riders` and `/ws/drivers` (endpoints work).
- Backend logs showed intermittent `connection not found` and empty Jaeger endpoint noise (`Post "": unsupported protocol scheme ""` when `JAEGER_ENDPOINT=""`).

### Causes
1. **Timing:** WS failed while backend target was still unhealthy (before health + SG fixes).
2. **ALB idle timeout:** default **60s** idle timeout drops long-lived map sockets if no traffic/pings.

### Fix
- Hard-refresh the map after backend is healthy.
- Raised ALB idle timeout via Ingress annotation:

  ```yaml
  alb.ingress.kubernetes.io/load-balancer-attributes: idle_timeout.timeout_seconds=3600
  ```

  in `gitops/cloudpulse/values-production.yaml`.

### Optional follow-ups
- Add WebSocket ping/pong in api-gateway and/or client.
- Skip Jaeger init when `JAEGER_ENDPOINT` is empty.

---

## Quick reference — current healthy shape

| Layer | Status after fixes |
|-------|--------------------|
| EKS nodes | Ready (2×) |
| Argo CD | Running; Application Synced |
| Frontend | Running; ALB TG healthy |
| API gateway | Running; `/health` → 200; ALB TG healthy |
| trip / driver / payment | Images on ECR; pods Running |
| Mongo / RabbitMQ | Running in-cluster |
| DNS | Alias → ALB for apex / api / www |
| TLS | ACM cert on ALB |
| WebSocket | `wss://api.cloudpulse.live/ws/...` upgrades OK |

**URLs**

- App: https://cloudpulse.live  
- API: https://api.cloudpulse.live  

---

## Related paths in the repo

| Area | Path |
|------|------|
| Terraform root | `terraform/` |
| DNS module | `terraform/modules/dns/` |
| EKS add-ons | `terraform/modules/eks-addons/` |
| Helm chart | `charts/cloudpulse/` |
| GitOps values | `gitops/cloudpulse/values-production.yaml` |
| Argo Application | `gitops/argocd/application.yaml` |
| CI | `.github/workflows/ci-cd.yaml` |
