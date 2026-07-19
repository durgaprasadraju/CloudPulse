locals {
  tags = merge(var.tags, {
    Project     = var.project_name
    Environment = var.environment
  })

  # Must match modules/eks local.cluster_name
  cluster_name = "${var.project_name}-${var.environment}"
}

# ---------------------------------------------------------------------------
# Networking
# ---------------------------------------------------------------------------
module "vpc" {
  source = "./modules/vpc"

  project_name = var.project_name
  environment  = var.environment
  cluster_name = local.cluster_name
  vpc_cidr     = var.vpc_cidr
  az_count     = var.az_count
  tags         = local.tags
}

# ---------------------------------------------------------------------------
# Compute — EKS 1.30 (nodes in private subnets, egress via NAT)
# ---------------------------------------------------------------------------
module "eks" {
  source = "./modules/eks"

  project_name           = var.project_name
  environment            = var.environment
  vpc_id                 = module.vpc.vpc_id
  subnet_ids             = concat(module.vpc.private_subnet_ids, module.vpc.public_subnet_ids)
  node_subnet_ids        = module.vpc.private_subnet_ids
  kubernetes_version     = var.kubernetes_version
  endpoint_public_access = var.eks_endpoint_public_access
  node_instance_types    = var.node_instance_types
  node_desired_size      = var.node_desired_size
  node_min_size          = var.node_min_size
  node_max_size          = var.node_max_size
  tags                   = local.tags
}

# ---------------------------------------------------------------------------
# Data — PostgreSQL (driver / profile data)
# ---------------------------------------------------------------------------
module "rds" {
  source = "./modules/rds"

  project_name = var.project_name
  environment  = var.environment
  vpc_id       = module.vpc.vpc_id
  subnet_ids   = module.vpc.private_subnet_ids

  # Allow both the custom node SG (launch template) and the EKS primary cluster SG,
  # plus the VPC CIDR as a safe fallback for pod networking edge cases.
  # Static map keys so for_each works when SG IDs are unknown until apply.
  allowed_security_group_ids = {
    eks_nodes           = module.eks.node_security_group_id
    eks_cluster_primary = module.eks.cluster_primary_security_group_id
  }
  allowed_cidr_blocks = [module.vpc.vpc_cidr]

  instance_class      = var.rds_instance_class
  engine_version      = var.rds_engine_version
  db_name             = var.rds_db_name
  multi_az            = var.rds_multi_az
  deletion_protection = var.rds_deletion_protection
  tags                = local.tags
}

# ---------------------------------------------------------------------------
# Data — Redis (real-time tracking)
# ---------------------------------------------------------------------------
module "elasticache" {
  source = "./modules/elasticache"

  project_name = var.project_name
  environment  = var.environment
  vpc_id       = module.vpc.vpc_id
  subnet_ids   = module.vpc.private_subnet_ids

  # Static map keys so for_each works when SG IDs are unknown until apply.
  allowed_security_group_ids = {
    eks_nodes           = module.eks.node_security_group_id
    eks_cluster_primary = module.eks.cluster_primary_security_group_id
  }
  allowed_cidr_blocks = [module.vpc.vpc_cidr]

  node_type                  = var.redis_node_type
  engine_version             = var.redis_engine_version
  num_cache_clusters         = var.redis_num_cache_clusters
  transit_encryption_enabled = var.redis_transit_encryption_enabled
  tags                       = local.tags
}

# ---------------------------------------------------------------------------
# DNS — Route 53 + ACM for cloudpulse.live / api.cloudpulse.live
# ---------------------------------------------------------------------------
module "dns" {
  source = "./modules/dns"

  domain_name        = var.domain_name
  create_hosted_zone = var.create_hosted_zone
  hosted_zone_id     = var.hosted_zone_id

  # Leave empty on first apply. After ALB exists (Ingress), set:
  #   alb_dns_name = "k8s-....elb.amazonaws.com"
  #   alb_zone_id  = "Z...."   # ALB's Route 53 hosted zone ID (not your domain zone)
  alb_dns_name = var.alb_dns_name
  alb_zone_id  = var.alb_zone_id

  tags = local.tags
}

# ---------------------------------------------------------------------------
# EKS add-ons — Helm installs (ALB Controller, metrics-server, Argo CD)
# ---------------------------------------------------------------------------
module "eks_addons" {
  source = "./modules/eks-addons"

  aws_region        = var.aws_region
  cluster_name      = module.eks.cluster_name
  vpc_id            = module.vpc.vpc_id
  oidc_provider_arn = module.eks.oidc_provider_arn
  oidc_provider_url = module.eks.oidc_provider_url

  enable_metrics_server               = var.enable_metrics_server
  enable_aws_load_balancer_controller = var.enable_aws_load_balancer_controller
  enable_argocd                       = var.enable_argocd
  bootstrap_cloudpulse_app            = var.bootstrap_cloudpulse_app
  argocd_server_service_type          = var.argocd_server_service_type

  argocd_app_project_manifest = "${path.module}/../gitops/argocd/application-project.yaml"
  argocd_application_manifest = "${path.module}/../gitops/argocd/application.yaml"

  tags = local.tags

  depends_on = [module.eks]
}
