# CloudPulse Route 53 + ACM (cloudpulse.live / api / www)

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
  apex_domain = var.domain_name
  # api + www + any extra SANs, always including the apex as the primary name
  certificate_sans = distinct(concat(
    ["api.${var.domain_name}", "www.${var.domain_name}"],
    var.extra_subject_alternative_names,
  ))
}

# ---------------------------------------------------------------------------
# Route 53 hosted zone
# ---------------------------------------------------------------------------
resource "aws_route53_zone" "this" {
  count = var.create_hosted_zone ? 1 : 0

  name = local.apex_domain

  tags = merge(var.tags, {
    Name = local.apex_domain
  })
}

data "aws_route53_zone" "by_id" {
  count   = !var.create_hosted_zone && var.hosted_zone_id != "" ? 1 : 0
  zone_id = var.hosted_zone_id
}

data "aws_route53_zone" "by_name" {
  count        = !var.create_hosted_zone && var.hosted_zone_id == "" ? 1 : 0
  name         = local.apex_domain
  private_zone = false
}

locals {
  zone_id = (
    var.create_hosted_zone
    ? aws_route53_zone.this[0].zone_id
    : (var.hosted_zone_id != "" ? data.aws_route53_zone.by_id[0].zone_id : data.aws_route53_zone.by_name[0].zone_id)
  )
  zone_arn = (
    var.create_hosted_zone
    ? aws_route53_zone.this[0].arn
    : (var.hosted_zone_id != "" ? data.aws_route53_zone.by_id[0].arn : data.aws_route53_zone.by_name[0].arn)
  )
  name_servers = (
    var.create_hosted_zone
    ? aws_route53_zone.this[0].name_servers
    : (var.hosted_zone_id != "" ? data.aws_route53_zone.by_id[0].name_servers : data.aws_route53_zone.by_name[0].name_servers)
  )
}

# ---------------------------------------------------------------------------
# ACM certificate (same region as ALB) + DNS validation
# ---------------------------------------------------------------------------
resource "aws_acm_certificate" "this" {
  domain_name               = local.apex_domain
  subject_alternative_names = local.certificate_sans
  validation_method         = "DNS"

  lifecycle {
    create_before_destroy = true
  }

  tags = merge(var.tags, {
    Name = local.apex_domain
  })
}

resource "aws_route53_record" "cert_validation" {
  for_each = {
    for dvo in aws_acm_certificate.this.domain_validation_options : dvo.domain_name => {
      name   = dvo.resource_record_name
      record = dvo.resource_record_value
      type   = dvo.resource_record_type
    }
  }

  allow_overwrite = true
  name            = each.value.name
  records         = [each.value.record]
  ttl             = 60
  type            = each.value.type
  zone_id         = local.zone_id
}

resource "aws_acm_certificate_validation" "this" {
  certificate_arn         = aws_acm_certificate.this.arn
  validation_record_fqdns = [for r in aws_route53_record.cert_validation : r.fqdn]
}

# ---------------------------------------------------------------------------
# ALB alias records (optional — set after Ingress creates the ALB)
# ---------------------------------------------------------------------------
resource "aws_route53_record" "apex" {
  count = var.alb_dns_name != "" && var.alb_zone_id != "" ? 1 : 0

  zone_id = local.zone_id
  name    = local.apex_domain
  type    = "A"

  alias {
    name                   = var.alb_dns_name
    zone_id                = var.alb_zone_id
    evaluate_target_health = true
  }
}

resource "aws_route53_record" "api" {
  count = var.alb_dns_name != "" && var.alb_zone_id != "" ? 1 : 0

  zone_id = local.zone_id
  name    = "api.${local.apex_domain}"
  type    = "A"

  alias {
    name                   = var.alb_dns_name
    zone_id                = var.alb_zone_id
    evaluate_target_health = true
  }
}

resource "aws_route53_record" "www" {
  count = var.alb_dns_name != "" && var.alb_zone_id != "" ? 1 : 0

  zone_id = local.zone_id
  name    = "www.${local.apex_domain}"
  type    = "A"

  alias {
    name                   = var.alb_dns_name
    zone_id                = var.alb_zone_id
    evaluate_target_health = true
  }
}
