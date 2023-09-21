resource "prodvana_k8s_runtime" "example" {
  name = "my-k8s-runtime"
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
}
