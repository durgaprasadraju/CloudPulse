output "hosted_zone_id" {
  description = "Route 53 hosted zone ID"
  value       = local.zone_id
}

output "hosted_zone_arn" {
  description = "Route 53 hosted zone ARN"
  value       = local.zone_arn
}

output "name_servers" {
  description = "NS records — set these at your domain registrar if Terraform created the zone"
  value       = local.name_servers
}

output "domain_name" {
  description = "Apex domain"
  value       = local.apex_domain
}

output "api_hostname" {
  description = "API hostname"
  value       = "api.${local.apex_domain}"
}

output "www_hostname" {
  description = "WWW hostname"
  value       = "www.${local.apex_domain}"
}

output "acm_certificate_arn" {
  description = "Validated ACM certificate ARN (use in ALB Ingress annotation)"
  value       = aws_acm_certificate_validation.this.certificate_arn
}

output "acm_certificate_status" {
  description = "ACM certificate domain name"
  value       = aws_acm_certificate.this.domain_name
}

output "alb_records_created" {
  description = "Whether ALB alias A records were created"
  value       = var.alb_dns_name != "" && var.alb_zone_id != ""
}
