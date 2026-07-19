variable "aws_region" {
  type = string
}

variable "cluster_name" {
  type = string
}

variable "vpc_id" {
  type = string
}

variable "oidc_provider_arn" {
  type = string
}

variable "oidc_provider_url" {
  description = "OIDC issuer URL without https://"
  type        = string
}

variable "enable_metrics_server" {
  type    = bool
  default = true
}

variable "enable_aws_load_balancer_controller" {
  type    = bool
  default = true
}

variable "enable_argocd" {
  type    = bool
  default = true
}

variable "bootstrap_cloudpulse_app" {
  description = "Apply gitops/argocd AppProject + Application after ArgoCD is installed"
  type        = bool
  default     = true
}

variable "metrics_server_chart_version" {
  type    = string
  default = "3.12.2"
}

variable "alb_controller_chart_version" {
  type    = string
  default = "1.11.0"
}

variable "argocd_chart_version" {
  type    = string
  default = "7.7.16"
}

variable "argocd_server_service_type" {
  description = "ClusterIP (port-forward) or LoadBalancer"
  type        = string
  default     = "ClusterIP"
}

variable "argocd_app_project_manifest" {
  type = string
}

variable "argocd_application_manifest" {
  type = string
}

variable "tags" {
  type    = map(string)
  default = {}
}
