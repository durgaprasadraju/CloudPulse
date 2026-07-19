# CloudPulse production Terraform

Provisions the AWS production foundation for **CloudPulse.live**:

| Module | What it creates |
|--------|-----------------|
| `modules/vpc` | VPC, 3 public + 3 private subnets, IGW, NAT Gateway, EKS/ALB subnet tags |
| `modules/eks` | EKS **1.30**, managed node group (`t3.medium`) in **private** subnets via **launch template** (node SG attached), **OIDC / IRSA** |
| `modules/rds` | PostgreSQL (Multi-AZ) in private subnets; credentials in **Secrets Manager** |
| `modules/elasticache` | Redis replication group in private subnets |
| `modules/dns` | Route 53 hosted zone + ACM cert for `cloudpulse.live`, `api.cloudpulse.live`, `www.cloudpulse.live` |

Private subnets route `0.0.0.0/0` through the NAT Gateway so EKS nodes can pull images and call AWS APIs without public IPs. RDS/Redis allow ingress from the EKS node SG, cluster primary SG, and VPC CIDR.

## Prerequisites

- Terraform `>= 1.5`
- AWS credentials with permissions for VPC, EKS, IAM, RDS, ElastiCache, Route 53, ACM
- Domain `cloudpulse.live` (or set `domain_name`) — if Terraform creates the hosted zone, point the registrar NS to `route53_name_servers`
- `aws` CLI (for `kubectl` config after apply)

## Usage

```bash
cd terraform
cp terraform.tfvars.example terraform.tfvars   # edit region / sizes as needed

terraform init
terraform plan
terraform apply
```

Configure kubectl:

```bash
terraform output -raw configure_kubectl | bash
```

## DNS / TLS

After apply:

1. If `create_hosted_zone = true`, set your registrar nameservers to:
   ```bash
   terraform output route53_name_servers
   ```
   ACM validation waits until those NS are live.
2. Put the cert on the ALB Ingress:
   ```bash
   terraform output -raw acm_certificate_arn
   ```
   Set in `gitops/cloudpulse/values-production.yaml`:
   ```yaml
   alb.ingress.kubernetes.io/certificate-arn: <arn>
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

Example trust principal:  
`system:serviceaccount:<namespace>:<service-account-name>`

## Notes

- RDS master password is generated and exposed (sensitive) via `rds_connection_url`.
- Prefer enabling the commented S3 backend in `provider.tf` before team use.
- Destroying production RDS requires `rds_deletion_protection = false` first.
