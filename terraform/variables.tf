# ---------------------------------------------------------------------------
# Global
# ---------------------------------------------------------------------------
variable "project_name" {
  description = "Project name used for naming AWS resources"
  type        = string
  default     = "cloudpulse"
}

variable "environment" {
  description = "Deployment environment name"
  type        = string
  default     = "production"
}

variable "aws_region" {
  description = "AWS region for the production stack"
  type        = string
  default     = "us-east-1"
}

variable "tags" {
  description = "Extra tags applied to all modules"
  type        = map(string)
  default     = {}
}

# ---------------------------------------------------------------------------
# VPC
# ---------------------------------------------------------------------------
variable "vpc_cidr" {
  description = "CIDR block for the production VPC"
  type        = string
  default     = "10.0.0.0/16"
}

variable "az_count" {
  description = "Number of Availability Zones (fixed to 3 for production)"
  type        = number
  default     = 3
}

# ---------------------------------------------------------------------------
# EKS
# ---------------------------------------------------------------------------
variable "kubernetes_version" {
  description = "EKS Kubernetes version"
  type        = string
  default     = "1.30"
}

variable "eks_endpoint_public_access" {
  description = "Expose the EKS API endpoint publicly (still requires IAM auth)"
  type        = bool
  default     = true
}

variable "node_instance_types" {
  description = "Instance types for the EKS managed node group"
  type        = list(string)
  default     = ["t3.medium"]
}

variable "node_desired_size" {
  description = "Desired worker node count"
  type        = number
  default     = 2
}

variable "node_min_size" {
  description = "Minimum worker node count"
  type        = number
  default     = 2
}

variable "node_max_size" {
  description = "Maximum worker node count"
  type        = number
  default     = 4
}

# ---------------------------------------------------------------------------
# RDS PostgreSQL
# ---------------------------------------------------------------------------
variable "rds_instance_class" {
  description = "RDS PostgreSQL instance class"
  type        = string
  default     = "db.t3.medium"
}

variable "rds_engine_version" {
  description = "PostgreSQL engine version (major or major.minor as supported in the region)"
  type        = string
  default     = "16"
}

variable "rds_db_name" {
  description = "Initial PostgreSQL database name"
  type        = string
  default     = "cloudpulse"
}

variable "rds_multi_az" {
  description = "Enable Multi-AZ for RDS"
  type        = bool
  default     = true
}

variable "rds_deletion_protection" {
  description = "Enable deletion protection on RDS"
  type        = bool
  default     = true
}

# ---------------------------------------------------------------------------
# ElastiCache Redis
# ---------------------------------------------------------------------------
variable "redis_node_type" {
  description = "ElastiCache Redis node type"
  type        = string
  default     = "cache.t3.medium"
}

variable "redis_engine_version" {
  description = "Redis engine version"
  type        = string
  default     = "7.1"
}

variable "redis_num_cache_clusters" {
  description = "Number of Redis nodes (primary + replicas)"
  type        = number
  default     = 2
}

variable "redis_transit_encryption_enabled" {
  description = "Enable Redis TLS (clients must use rediss:// when true)"
  type        = bool
  default     = false
}
