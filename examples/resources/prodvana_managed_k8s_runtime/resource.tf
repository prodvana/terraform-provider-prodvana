resource "prodvana_managed_k8s_runtime" "example" {
  name = "my-k8s-runtime"
  agent_env = {
    "PROXY" = "http://localhost:8080"
  }

  labels = [
    {
      label = "env"
      value = "staging"
    },
    {
      label = "region"
      value = "us-central1"
    },
  ]

  config_path    = "~/.kube/config"
  config_context = "my-k8s-context"
}
