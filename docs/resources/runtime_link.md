---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "prodvana_runtime_link Resource - terraform-provider-prodvana"
subcategory: ""
description: |-
  (Alpha! This feature is still in progress.)
  A runtime_link resource represents a successfully linked runtime.
  This is most useful for Kubernetes runtimes -- the agent must be installed and registered with the Prodvana service before the runtime can be used.
  Pair this with an explicit depends_on block ensures that the runtime is ready before attempting to use it. See the example below.
---

# prodvana_runtime_link (Resource)

(Alpha! This feature is still in progress.) 
A `runtime_link` resource represents a successfully linked runtime.
This is most useful for Kubernetes runtimes -- the agent must be installed and registered with the Prodvana service before the runtime can be used.
Pair this with an explicit `depends_on` block ensures that the runtime is ready before attempting to use it. See the example below.

## Example Usage

```terraform
# create the runtime placeholder in Prodvana
resource "prodvana_runtime" "example" {
  name = "my-runtime"
  type = "K8S"
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
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) Name of the runtime to wait for linking.

### Optional

- `timeout` (String) How long to wait for the runtime linking to complete. A valid Go duration string, e.g. `10m` or `1h`. Defaults to `10m`

### Read-Only

- `id` (String) Runtime identifier

