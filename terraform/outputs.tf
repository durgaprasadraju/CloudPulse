output "vpc_id" {
  description = "Production VPC ID"
  value       = module.vpc.vpc_id
}

output "vpc_cidr" {
  description = "Production VPC CIDR"
  value       = module.vpc.vpc_cidr
}

output "private_subnet_ids" {
  description = "Private subnet IDs (EKS nodes, RDS, Redis)"
  value       = module.vpc.private_subnet_ids
}

output "public_subnet_ids" {
  description = "Public subnet IDs (NAT / load balancers)"
  value       = module.vpc.public_subnet_ids
}

output "eks_cluster_name" {
  description = "EKS cluster name"
  value       = module.eks.cluster_name
}

output "eks_cluster_endpoint" {
  description = "EKS API endpoint"
  value       = module.eks.cluster_endpoint
}

output "eks_cluster_version" {
  description = "EKS Kubernetes version"
  value       = module.eks.cluster_version
}

output "eks_oidc_provider_arn" {
  description = "OIDC provider ARN for IRSA"
  value       = module.eks.oidc_provider_arn
}

output "eks_oidc_provider_url" {
  description = "OIDC issuer URL (without https://) for IRSA trust policies"
  value       = module.eks.oidc_provider_url
}

output "eks_node_security_group_id" {
  description = "Custom node security group (launch template)"
  value       = module.eks.node_security_group_id
}

output "eks_cluster_primary_security_group_id" {
  description = "EKS primary cluster security group"
  value       = module.eks.cluster_primary_security_group_id
}

output "rds_endpoint" {
  description = "PostgreSQL endpoint"
  value       = module.rds.db_endpoint
}

output "rds_connection_url" {
  description = "PostgreSQL connection URL"
  value       = module.rds.connection_url
  sensitive   = true
}

output "rds_secrets_manager_arn" {
  description = "Secrets Manager ARN for RDS credentials"
  value       = module.rds.secrets_manager_arn
}

output "redis_primary_endpoint" {
  description = "Redis primary endpoint"
  value       = module.elasticache.primary_endpoint_address
}

output "redis_url" {
  description = "Redis connection URL"
  value       = module.elasticache.redis_url
}

output "configure_kubectl" {
  description = "Command to configure kubectl for this cluster"
  value       = "aws eks update-kubeconfig --region ${var.aws_region} --name ${module.eks.cluster_name}"
}
