# CloudPulse production Terraform

Provisions the AWS production foundation for **CloudPulse.live**:

| Module | What it creates |
|--------|-----------------|
| `modules/vpc` | VPC, 3 public + 3 private subnets, IGW, NAT Gateway, EKS/ALB subnet tags |
| `modules/eks` | EKS **1.30**, managed node group (`t3.medium`) in **private** subnets via **launch template** (node SG attached), **OIDC / IRSA** |
| `modules/rds` | PostgreSQL (Multi-AZ) in private subnets; credentials in **Secrets Manager** |
| `modules/elasticache` | Redis replication group in private subnets |
| `modules/dns` | Route 53 hosted zone + ACM cert for `cloudpulse.live`, `api.cloudpulse.live`, `www.cloudpulse.live` |
| `modules/eks-addons` | **Helm** installs: metrics-server, AWS Load Balancer Controller (IRSA), **Argo CD**, plus CloudPulse AppProject/Application |

Private subnets route `0.0.0.0/0` through the NAT Gateway so EKS nodes can pull images and call AWS APIs without public IPs. RDS/Redis allow ingress from the EKS node SG, cluster primary SG, and VPC CIDR.

## Prerequisites

- Terraform `>= 1.5`
- AWS credentials with permissions for VPC, EKS, IAM, RDS, ElastiCache, Route 53, ACM
- Domain `cloudpulse.live` (or set `domain_name`) — if Terraform creates the hosted zone, point the registrar NS to `route53_name_servers`
- `aws` CLI + `kubectl` (for Argo CD password / port-forward)
- Helm charts are installed **by Terraform** (you do not need to run `helm` manually)

## Usage

```bash
cd terraform
cp terraform.tfvars.example terraform.tfvars   # edit region / sizes as needed

terraform init
terraform plan
terraform apply
```

If this is the **first** cluster create, apply EKS before add-ons:

```bash
terraform apply -target=module.vpc -target=module.eks
terraform apply
```

### If ALB Controller was installed manually earlier

Either let Terraform take over the Helm release, or uninstall the CLI install first:

```bash
helm uninstall aws-load-balancer-controller -n kube-system
# optional: delete the old IAM role created by hand
terraform apply
```

If metrics-server was installed with raw YAML (not Helm), set in `terraform.tfvars`:

```hcl
enable_metrics_server = false
```

Configure kubectl:

```bash
terraform output -raw configure_kubectl | bash
```

### Argo CD UI

```bash
terraform output -raw argocd_port_forward | bash
# password:
terraform output -raw argocd_admin_password_cmd | bash
# login: admin / <password>  →  https://localhost:8080
```

Argo CD syncs `gitops/argocd/application.yaml` → Helm chart `charts/cloudpulse` + `gitops/cloudpulse/values-production.yaml`.

## DNS / TLS

After apply:

1. If `create_hosted_zone = true`, set your registrar nameservers to:
   ```bash
   terraform output route53_name_servers
   ```
   ACM validation waits until those NS are live.
2. Put the cert on the ALB Ingress (also set in GitOps values):
   ```bash
   terraform output -raw acm_certificate_arn
   ```
3. After AWS Load Balancer Controller creates the ALB, create alias records by setting in `terraform.tfvars`:
   ```hcl
   alb_dns_name = "k8s-cloudpulse-....elb.amazonaws.com"
   alb_zone_id  = "Z35SXDOTRQ7X7K"   # ALB's zone ID from describe-load-balancers
   ```
   Then `terraform apply` again — creates A aliases for apex, `api`, and `www`.

## IRSA

After apply, use these outputs when creating IAM roles for Kubernetes service accounts:

- `eks_oidc_provider_arn`
- `eks_oidc_provider_url`
- `alb_controller_role_arn` (Load Balancer Controller)

Example trust principal:  
`system:serviceaccount:<namespace>:<service-account-name>`

## Notes

- RDS master password is generated and exposed (sensitive) via `rds_connection_url`.
- Prefer enabling the commented S3 backend in `provider.tf` before team use.
- Destroying production RDS requires `rds_deletion_protection = false` first.
