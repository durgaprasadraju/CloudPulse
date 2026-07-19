# CloudPulse EKS add-ons installed via Terraform Helm provider:
# - AWS Load Balancer Controller (Ingress → ALB)
# - metrics-server (HPA)
# - ArgoCD (+ CloudPulse AppProject / Application)

terraform {
  required_version = ">= 1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0"
    }
    helm = {
      source  = "hashicorp/helm"
      version = ">= 2.12"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = ">= 2.25"
    }
    time = {
      source  = "hashicorp/time"
      version = ">= 0.11"
    }
    null = {
      source  = "hashicorp/null"
      version = ">= 3.2"
    }
  }
}

locals {
  alb_sa_namespace = "kube-system"
  alb_sa_name      = "aws-load-balancer-controller"
  oidc_issuer      = replace(var.oidc_provider_url, "https://", "")
}

# ---------------------------------------------------------------------------
# IRSA — AWS Load Balancer Controller
# ---------------------------------------------------------------------------
data "aws_iam_policy_document" "alb_controller_assume" {
  statement {
    actions = ["sts:AssumeRoleWithWebIdentity"]
    effect  = "Allow"

    principals {
      type        = "Federated"
      identifiers = [var.oidc_provider_arn]
    }

    condition {
      test     = "StringEquals"
      variable = "${local.oidc_issuer}:aud"
      values   = ["sts.amazonaws.com"]
    }

    condition {
      test     = "StringEquals"
      variable = "${local.oidc_issuer}:sub"
      values   = ["system:serviceaccount:${local.alb_sa_namespace}:${local.alb_sa_name}"]
    }
  }
}

resource "aws_iam_role" "alb_controller" {
  name               = "${var.cluster_name}-alb-controller"
  assume_role_policy = data.aws_iam_policy_document.alb_controller_assume.json
  tags               = var.tags
}

resource "aws_iam_policy" "alb_controller" {
  name   = "${var.cluster_name}-AWSLoadBalancerControllerIAMPolicy"
  policy = file("${path.module}/iam_policy_alb.json")
  tags   = var.tags
}

resource "aws_iam_role_policy_attachment" "alb_controller" {
  role       = aws_iam_role.alb_controller.name
  policy_arn = aws_iam_policy.alb_controller.arn
}

# ---------------------------------------------------------------------------
# metrics-server (required for HPA)
# ---------------------------------------------------------------------------
resource "helm_release" "metrics_server" {
  count = var.enable_metrics_server ? 1 : 0

  name       = "metrics-server"
  repository = "https://kubernetes-sigs.github.io/metrics-server/"
  chart      = "metrics-server"
  namespace  = "kube-system"
  version    = var.metrics_server_chart_version

  values = [yamlencode({
    args = ["--kubelet-insecure-tls"]
  })]
}

# ---------------------------------------------------------------------------
# AWS Load Balancer Controller
# ---------------------------------------------------------------------------
resource "helm_release" "aws_load_balancer_controller" {
  count = var.enable_aws_load_balancer_controller ? 1 : 0

  name       = "aws-load-balancer-controller"
  repository = "https://aws.github.io/eks-charts"
  chart      = "aws-load-balancer-controller"
  namespace  = local.alb_sa_namespace
  version    = var.alb_controller_chart_version

  create_namespace = false

  values = [yamlencode({
    clusterName = var.cluster_name
    region      = var.aws_region
    vpcId       = var.vpc_id
    serviceAccount = {
      create = true
      name   = local.alb_sa_name
      annotations = {
        "eks.amazonaws.com/role-arn" = aws_iam_role.alb_controller.arn
      }
    }
  })]

  depends_on = [
    aws_iam_role_policy_attachment.alb_controller,
    helm_release.metrics_server,
  ]
}

# ---------------------------------------------------------------------------
# Argo CD
# ---------------------------------------------------------------------------
resource "helm_release" "argocd" {
  count = var.enable_argocd ? 1 : 0

  name             = "argocd"
  repository       = "https://argoproj.github.io/argo-helm"
  chart            = "argo-cd"
  namespace        = "argocd"
  version          = var.argocd_chart_version
  create_namespace = true

  # Keep bootstrap simple; expose server via port-forward or later Ingress
  values = [yamlencode({
    configs = {
      params = {
        "server.insecure" = true
      }
    }
    server = {
      service = {
        type = var.argocd_server_service_type
      }
    }
    dex = {
      enabled = false
    }
  })]

  depends_on = [helm_release.metrics_server]
}

resource "time_sleep" "wait_for_argocd_crds" {
  count = var.enable_argocd && var.bootstrap_cloudpulse_app ? 1 : 0

  depends_on      = [helm_release.argocd]
  create_duration = "45s"
}

# Bootstrap AppProject + Application after Argo CD CRDs exist.
# Use kubectl (not kubernetes_manifest) so plan does not require CRDs upfront.
resource "null_resource" "argocd_bootstrap" {
  count = var.enable_argocd && var.bootstrap_cloudpulse_app ? 1 : 0

  triggers = {
    project_sha = filesha256(var.argocd_app_project_manifest)
    app_sha     = filesha256(var.argocd_application_manifest)
    cluster     = var.cluster_name
  }

  provisioner "local-exec" {
    interpreter = ["/bin/bash", "-c"]
    command     = <<-EOT
      set -euo pipefail
      aws eks update-kubeconfig --name "${var.cluster_name}" --region "${var.aws_region}" --kubeconfig /tmp/cloudpulse-eks.kubeconfig
      export KUBECONFIG=/tmp/cloudpulse-eks.kubeconfig
      kubectl wait --for=condition=Established crd/appprojects.argoproj.io --timeout=120s
      kubectl wait --for=condition=Established crd/applications.argoproj.io --timeout=120s
      kubectl apply -f "${var.argocd_app_project_manifest}"
      kubectl apply -f "${var.argocd_application_manifest}"
    EOT
  }

  depends_on = [time_sleep.wait_for_argocd_crds]
}
