variable "project_name" {
  description = "Project name used for resource naming"
  type        = string
}

variable "environment" {
  description = "Deployment environment"
  type        = string
}

variable "vpc_id" {
  description = "VPC ID"
  type        = string
}

variable "subnet_ids" {
  description = "Private subnet IDs for the ElastiCache subnet group"
  type        = list(string)
}

variable "allowed_security_group_ids" {
  description = "Map of static key → security group ID allowed to connect to Redis (e.g. EKS nodes). Keys must be known at plan time."
  type        = map(string)
  default     = {}
}

variable "allowed_cidr_blocks" {
  description = "Optional CIDR blocks allowed to connect to Redis"
  type        = list(string)
  default     = []
}

variable "engine_version" {
  description = "Redis engine version"
  type        = string
  default     = "7.1"
}

variable "node_type" {
  description = "ElastiCache node type"
  type        = string
  default     = "cache.t3.medium"
}

variable "num_cache_clusters" {
  description = "Number of cache clusters (primary + replicas)"
  type        = number
  default     = 2
}

variable "parameter_group_name" {
  description = "Redis parameter group"
  type        = string
  default     = "default.redis7"
}

variable "transit_encryption_enabled" {
  description = "Enable in-transit encryption (TLS). When true, clients must use rediss://"
  type        = bool
  default     = false
}

variable "multi_az_enabled" {
  description = "Enable Multi-AZ (requires num_cache_clusters > 1)"
  type        = bool
  default     = true
}

variable "apply_immediately" {
  description = "Apply modifications immediately"
  type        = bool
  default     = false
}

variable "tags" {
  description = "Additional tags"
  type        = map(string)
  default     = {}
}
