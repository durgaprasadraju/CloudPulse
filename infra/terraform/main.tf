# Copyright 2026 Durga Prasad Raju Nadimpalli
# Licensed under the Apache License, Version 2.0

terraform {
  required_version = ">= 1.6.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

# Stub — wire EKS, RDS, ElastiCache in follow-up modules
output "cloudpulse_stack" {
  value = "terraform stub — see infra/terraform/README.md"
}
