# CloudPulse ElastiCache Redis (private subnets, for real-time tracking)

terraform {
  required_version = ">= 1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0"
    }
  }
}

locals {
  common_tags = merge(var.tags, {
    Project     = var.project_name
    Environment = var.environment
    ManagedBy   = "terraform"
  })

  name = "${var.project_name}-${var.environment}-redis"
}

resource "aws_elasticache_subnet_group" "this" {
  name       = local.name
  subnet_ids = var.subnet_ids

  tags = merge(local.common_tags, {
    Name = local.name
  })
}

resource "aws_security_group" "this" {
  name_prefix = "${local.name}-"
  description = "Redis access from EKS / app security groups"
  vpc_id      = var.vpc_id

  tags = merge(local.common_tags, {
    Name = "${local.name}-sg"
  })

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_security_group_rule" "ingress_cidr" {
  count = length(var.allowed_cidr_blocks) > 0 ? 1 : 0

  type              = "ingress"
  from_port         = 6379
  to_port           = 6379
  protocol          = "tcp"
  cidr_blocks       = var.allowed_cidr_blocks
  security_group_id = aws_security_group.this.id
  description       = "Redis from allowed CIDRs"
}

resource "aws_security_group_rule" "ingress_sg" {
  for_each = toset(var.allowed_security_group_ids)

  type                     = "ingress"
  from_port                = 6379
  to_port                  = 6379
  protocol                 = "tcp"
  source_security_group_id = each.value
  security_group_id        = aws_security_group.this.id
  description              = "Redis from ${each.value}"
}

resource "aws_security_group_rule" "egress" {
  type              = "egress"
  from_port         = 0
  to_port           = 0
  protocol          = "-1"
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = aws_security_group.this.id
}

resource "aws_elasticache_replication_group" "this" {
  replication_group_id = local.name
  description          = "CloudPulse Redis for real-time tracking"

  engine               = "redis"
  engine_version       = var.engine_version
  node_type            = var.node_type
  num_cache_clusters   = var.num_cache_clusters
  port                 = 6379
  parameter_group_name = var.parameter_group_name

  subnet_group_name  = aws_elasticache_subnet_group.this.name
  security_group_ids = [aws_security_group.this.id]

  at_rest_encryption_enabled = true
  transit_encryption_enabled = var.transit_encryption_enabled
  automatic_failover_enabled = var.num_cache_clusters > 1
  multi_az_enabled           = var.num_cache_clusters > 1 && var.multi_az_enabled

  apply_immediately          = var.apply_immediately
  auto_minor_version_upgrade = true

  tags = merge(local.common_tags, {
    Name = local.name
  })
}
