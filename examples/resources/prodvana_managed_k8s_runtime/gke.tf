resource "google_container_cluster" "cluster" {
  name     = "my-gke-cluster"
  location = "us-central1"

  // <...configuration elided...>
}

data "google_client_config" "default" {}

resource "prodvana_managed_k8s_runtime" "example" {
  name = "my-k8s-runtime"
  agent_env = {
    "PROXY" = "http://localhost:8080"
  }

  host                   = google_container_cluster.cluster.endpoint
  cluster_ca_certificate = base64decode(google_container_cluster.test.master_auth.0.cluster_ca_certificate)
  token                  = data.google_client_config.default.access_token
}
