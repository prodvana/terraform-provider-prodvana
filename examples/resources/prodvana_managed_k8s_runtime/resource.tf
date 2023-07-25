// EKS Example

resource "aws_eks_cluster" "cluster" {
  name = "my-eks-cluster"

  // <...configuration elided...>
}

// Note: this requires the AWS provider be configured with IAM credentials that have access to the EKS cluster
data "aws_eks_cluster_auth" "cluster" {
  name = aws_eks_cluster.cluster.name
}

resource "prodvana_managed_k8s_runtime" "example" {
  name = "my-k8s-runtime"
  agent_env = {
    "PROXY" = "http://localhost:8080"
  }

  host                   = data.aws_eks_cluster_auth.cluster.endpoint
  cluster_ca_certificate = base64decode(data.aws_eks_cluster_auth.cluster.certificate_authority.0.data)
  token                  = data.aws_eks_cluster_auth.cluster.token
}

// GKE Example

resource "google_container_cluster" "cluster" {
  name     = "my-gke-cluster"
  location = "us-central1"

  // <...configuration elided...>
}

resource "prodvana_managed_k8s_runtime" "example" {
  name = "my-k8s-runtime"
  agent_env = {
    "PROXY" = "http://localhost:8080"
  }

  host                   = google_container_cluster.cluster.endpoint
  cluster_ca_certificate = google_container_cluster.cluster.master_auth.0.cluster_ca_certificate
  client_certificate     = google_container_cluster.cluster.master_auth.0.client_certificate
  client_key             = google_container_cluster.cluster.master_auth.0.client_key
}
