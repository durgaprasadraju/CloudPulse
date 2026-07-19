output "alb_controller_role_arn" {
  description = "IAM role ARN for the AWS Load Balancer Controller service account"
  value       = aws_iam_role.alb_controller.arn
}

output "argocd_namespace" {
  value = var.enable_argocd ? "argocd" : null
}

output "argocd_server_port_forward" {
  description = "Command to open the Argo CD UI locally"
  value       = var.enable_argocd ? "kubectl -n argocd port-forward svc/argocd-server 8080:443" : null
}

output "argocd_initial_admin_password_cmd" {
  description = "Command to read the initial Argo CD admin password"
  value       = var.enable_argocd ? "kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' | base64 -d; echo" : null
}
