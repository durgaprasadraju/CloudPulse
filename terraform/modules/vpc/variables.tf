variable "project_name" {
  description = "Project name used for resource naming"
  type        = string
}

variable "environment" {
  description = "Deployment environment (e.g. production)"
  type        = string
}

variable "cluster_name" {
  description = "EKS cluster name used for kubernetes.io/cluster/* subnet tags (must match EKS module naming)"
  type        = string
}

variable "vpc_cidr" {
  description = "CIDR block for the VPC"
  type        = string
  default     = "10.0.0.0/16"
}

variable "az_count" {
  description = "Number of Availability Zones (must be 3 for CloudPulse production)"
  type        = number
  default     = 3

  validation {
    condition     = var.az_count == 3
    error_message = "CloudPulse production VPC must span exactly 3 Availability Zones."
  }
}

variable "tags" {
  description = "Additional tags applied to all VPC resources"
  type        = map(string)
  default     = {}
}
