# CloudPulse Helm Chart

Deploys the CloudPulse.live production apps onto EKS:

| Component | Kubernetes resources |
|-----------|----------------------|
| **Go backend** (API Gateway) | Deployment, Service, **HPA (CPU)** |
| **React frontend** (Next.js) | Deployment, Service |
| **Ingress** | AWS Load Balancer Controller (ALB) |

## Host routing

| Host | Service |
|------|---------|
| `api.cloudpulse.live` | Go backend |
| `cloudpulse.live` | React frontend |
| `www.cloudpulse.live` | React frontend (optional) |

## Prerequisites

1. EKS cluster (see `/terraform`)
2. [AWS Load Balancer Controller](https://kubernetes-sigs.github.io/aws-load-balancer-controller/) installed
3. [metrics-server](https://github.com/kubernetes-sigs/metrics-server) (required for HPA)
4. Container images pushed to a registry your nodes can pull (ECR recommended)
5. ACM certificate for `cloudpulse.live` / `api.cloudpulse.live` (for HTTPS)

## Install

From the repository root:

```bash
# Install / upgrade into namespace cloudpulse
helm upgrade --install cloudpulse ./charts/cloudpulse \
  --namespace cloudpulse \
  --create-namespace \
  --set backend.image.repository=ACCOUNT.dkr.ecr.REGION.amazonaws.com/cloudpulse/api-gateway \
  --set backend.image.tag=latest \
  --set frontend.image.repository=ACCOUNT.dkr.ecr.REGION.amazonaws.com/cloudpulse/web \
  --set frontend.image.tag=latest \
  --set-string ingress.annotations."alb\.ingress\.kubernetes\.io/certificate-arn"=arn:aws:acm:REGION:ACCOUNT:certificate/UUID
```

Using a values file:

```bash
cp charts/cloudpulse/values.yaml charts/cloudpulse/values-prod.yaml
# edit image repos + certificate-arn

helm upgrade --install cloudpulse ./charts/cloudpulse \
  --namespace cloudpulse \
  --create-namespace \
  -f charts/cloudpulse/values-prod.yaml
```

## Useful commands

```bash
# Render manifests without applying
helm template cloudpulse ./charts/cloudpulse -n cloudpulse

# Status
helm status cloudpulse -n cloudpulse
kubectl get all,hpa,ingress -n cloudpulse

# ALB hostname (for DNS)
kubectl get ingress -n cloudpulse -o wide

# Uninstall
helm uninstall cloudpulse -n cloudpulse
```

## Notes

- Backend HPA scales on **CPU utilization** (default target 70%, min 2 / max 10).
- Frontend `NEXT_PUBLIC_*` URLs should match public hosts; rebuild the web image if those values are compile-time only.
- Set IRSA on `serviceAccount.annotations` when pods need AWS API access.
