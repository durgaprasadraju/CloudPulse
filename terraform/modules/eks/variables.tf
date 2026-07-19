variable "project_name" {
  description = "Project name used for resource naming"
  type        = string
}

variable "environment" {
  description = "Deployment environment (e.g. production)"
  type        = string
}

variable "vpc_id" {
  description = "VPC ID hosting the EKS cluster"
  type        = string
}

variable "subnet_ids" {
  description = "Subnet IDs for the EKS control plane ENIs (typically private + public or private only)"
  type        = list(string)
}

variable "node_subnet_ids" {
  description = "Private subnet IDs for the managed node group"
  type        = list(string)
}

variable "kubernetes_version" {
  description = "EKS Kubernetes version"
  type        = string
  default     = "1.30"
}

variable "endpoint_public_access" {
  description = "Whether the EKS API endpoint is publicly reachable"
  type        = bool
  default     = true
}

variable "cluster_log_types" {
  description = "Control plane log types to enable"
  type        = list(string)
  default     = ["api", "audit", "authenticator"]
}

variable "node_instance_types" {
  description = "EC2 instance types for the managed node group"
  type        = list(string)
  default     = ["t3.medium"]
}

variable "node_capacity_type" {
  description = "ON_DEMAND or SPOT"
  type        = string
  default     = "ON_DEMAND"
}

variable "node_desired_size" {
  description = "Desired number of worker nodes"
  type        = number
  default     = 2
}

variable "node_min_size" {
  description = "Minimum number of worker nodes"
  type        = number
  default     = 2
}

variable "node_max_size" {
  description = "Maximum number of worker nodes"
  type        = number
  default     = 4
}

variable "node_disk_size" {
  description = "Root volume size (GiB) for worker nodes"
  type        = number
  default     = 50
}

variable "tags" {
  description = "Additional tags"
  type        = map(string)
  default     = {}
}
