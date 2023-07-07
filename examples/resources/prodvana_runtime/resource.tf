resource "prodvana_runtime" "example" {
  name = "my-runtime"
  type = "K8S"
  k8s = {
    agent_env = {
      "PROXY" = "example.com:8080"
    }
  }
}
