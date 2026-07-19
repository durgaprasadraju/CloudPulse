variable "domain_name" {
  description = "Apex domain (e.g. cloudpulse.live)"
  type        = string
}

variable "extra_subject_alternative_names" {
  description = "Additional SANs beyond api.<domain> and www.<domain>"
  type        = list(string)
  default     = []
}

variable "create_hosted_zone" {
  description = "Create a new public Route 53 hosted zone for the domain"
  type        = bool
  default     = true
}

variable "hosted_zone_id" {
  description = "Existing hosted zone ID when create_hosted_zone is false (optional if zone can be looked up by domain name)"
  type        = string
  default     = ""
}

variable "alb_dns_name" {
  description = "ALB DNS name for alias records (leave empty until Ingress creates the ALB)"
  type        = string
  default     = ""
}

variable "alb_zone_id" {
  description = "ALB hosted zone ID for alias records (leave empty until Ingress creates the ALB)"
  type        = string
  default     = ""
}

variable "tags" {
  description = "Additional tags"
  type        = map(string)
  default     = {}
}
