# create the runtime placeholder in Prodvana
resource "prodvana_k8s_runtime" "example" {
  name = "my-runtime"
}

# <...Full Kubernetes setup elided...>

# deploy the Prodvana agent to the Kubernetes cluster
# NOTE: this is an example and may not be complete, see
# https://docs.prodvana.io for the latest agent configuration details
resource "kubernetes_deployment_v1" "agent" {
  metadata {
    name      = "agent"
    namespace = "prodvana"
  }

  spec {
    replicas = 1
    template {
      spec {
        container {
          name  = "prodvana-agent"
          image = "prodvana/agent:v0.1.0"

          args = [
            "/agent",
            "--clusterid",
            prodvana_runtime.example.id,
            "--auth",
            prodvana_runtime.agent_api_token,
            "--server-addr",
            "api.<org_slug>.prodvana.io",
          ]

        }
      }
    }
  }
}

# this resource will complete only after the agent
# registers itself with the Prodvana API
resource "prodvana_runtime_link" "example" {
  id = prodvana_runtime.example.id
}

resource "prodvana_release_channel" "example" {
  name = "my-release-channel"
  runtimes = [
    {
      # now you can reference the runtime from the runtime_link resource
      # and be sure that the runtime has been fully linked prior to use
      runtime = prodvana_runtime_link.example.name
    }
  ]
}

