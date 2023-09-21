# create the runtime placeholder in Prodvana
resource "prodvana_k8s_runtime" "example" {
  name = "my-runtime"
}

# <...Full Kubernetes setup elided...>

# deploy the Prodvana agent to the Kubernetes cluster
# NOTE: this is an example and may not be complete, see
# https://docs.prodvana.io for the latest agent configuration details
resource "kubernetes_namespace" "prodvana" {
  metadata {
    name = "prodvana"
  }
}

resource "kubernetes_service_account" "prodvana" {
  metadata {
    name      = "prodvana"
    namespace = kubernetes_namespace.prodvana.metadata[0].name
  }
}

resource "kubernetes_cluster_role_binding" "prodvana" {
  metadata {
    name = "prodvana-access"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = "cluster-admin"
  }

  subject {
    kind      = "ServiceAccount"
    name      = "prodvana"
    namespace = kubernetes_namespace.prodvana.metadata[0].name
  }
}

resource "kubernetes_deployment" "agent" {
  metadata {
    name      = "prodvana-agent"
    namespace = kubernetes_namespace.prodvana.metadata[0].name
  }

  spec {
    replicas = 1

    selector {
      match_labels = {
        app = "prodvana-agent"
      }
    }

    template {
      metadata {
        labels = {
          app = "prodvana-agent"
        }
      }

      spec {
        service_account_name = kubernetes_service_account.prodvana.metadata[0].name
        container {
          name  = "agent"
          image = resource.prodvana_k8s_runtime.cluster.agent_image
          args  = resource.prodvana_k8s_runtime.cluster.agent_args
        }
      }
    }
  }
}

# this resource will complete only after the agent
# registers itself with the Prodvana API
resource "prodvana_runtime_link" "example" {
  id = prodvana_k8s_runtime.example.id
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

