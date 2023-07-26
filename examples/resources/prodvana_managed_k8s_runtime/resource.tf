resource "prodvana_managed_k8s_runtime" "example" {
  name = "my-k8s-runtime"
  agent_env = {
    "PROXY" = "http://localhost:8080"
  }

  config_path    = "~/.kube/config"
  config_context = "my-k8s-context"
}
