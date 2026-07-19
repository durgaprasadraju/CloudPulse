output "cluster_name" {
  description = "EKS cluster name"
  value       = aws_eks_cluster.this.name
}

output "cluster_endpoint" {
  description = "EKS API server endpoint"
  value       = aws_eks_cluster.this.endpoint
}

output "cluster_certificate_authority_data" {
  description = "Base64 encoded certificate data for the cluster"
  value       = aws_eks_cluster.this.certificate_authority[0].data
  sensitive   = true
}

output "cluster_security_group_id" {
  description = "Additional security group ID attached to the EKS control plane"
  value       = aws_security_group.cluster.id
}

output "cluster_primary_security_group_id" {
  description = "EKS-created cluster security group (auto-associated with managed nodes)"
  value       = aws_eks_cluster.this.vpc_config[0].cluster_security_group_id
}

output "node_security_group_id" {
  description = "Custom security group ID attached to worker nodes via launch template"
  value       = aws_security_group.node.id
}

output "node_role_arn" {
  description = "IAM role ARN used by the managed node group"
  value       = aws_iam_role.node.arn
}

output "oidc_provider_arn" {
  description = "IAM OIDC provider ARN for IRSA"
  value       = aws_iam_openid_connect_provider.this.arn
}

output "oidc_provider_url" {
  description = "OIDC issuer URL (without https://) for IRSA trust policies"
  value       = replace(aws_iam_openid_connect_provider.this.url, "https://", "")
}

output "cluster_version" {
  description = "Kubernetes version of the cluster"
  value       = aws_eks_cluster.this.version
}
