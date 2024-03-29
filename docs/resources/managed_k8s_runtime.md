---
page_title: "prodvana_managed_k8s_runtime Resource - terraform-provider-prodvana"
subcategory: ""
description: |-
  (Alpha! This feature is still in progress.) Manages a Kubernetes Runtime https://docs.prodvana.io/docs/prodvana-concepts#runtime.
  This resource links a Kubernetes runtime with Prodvana and fully manages the agent lifecycle.
  The agent will be installed as a Kubernetes deployment in the specified namespace, by this resource. After the initial agent install, Prodvana will manage the agent lifecycle, including upgrades, outside of Terraform.
---

# prodvana_managed_k8s_runtime (Resource)

(Alpha! This feature is still in progress.) Manages a Kubernetes [Runtime](https://docs.prodvana.io/docs/prodvana-concepts#runtime).
This resource links a Kubernetes runtime with Prodvana and fully manages the agent lifecycle.

The agent will be installed as a Kubernetes deployment in the specified namespace, *by this resource*. After the initial agent install, Prodvana will manage the agent lifecycle, including upgrades, outside of Terraform.

## Example Usage

Here's a simple example that links a runtime, assuming a local kubeconfig:

```terraform
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
```

Here's an an example of using `managed_k8s_runtime` with a Terraform created GKE cluster:

```terraform
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
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) Runtime name

### Optional

- `agent_env` (Map of String) Environment variables to pass to the agent. Useful for cases like passing proxy configuration to the agent if needed.
- `client_certificate` (String) PEM-encoded client certificate for TLS authentication.
- `client_key` (String) PEM-encoded client certificate key for TLS authentication.
- `cluster_ca_certificate` (String) PEM-encoded root certificates bundle for TLS authentication.
- `config_context` (String) Context to use from the kube config file.
- `config_context_auth_info` (String) Authentication info context of the kube config (name of the kubeconfig user, `--user` flag in `kubectl`).
- `config_context_cluster` (String) Cluster context of the kube config (name of the kubeconfig cluster, `--cluster` flag in `kubectl`).
- `config_path` (String) Path to the kube config file.
- `config_paths` (List of String) A list of paths to kube config files.
- `exec` (Attributes) Exec configuration for authentication to the Kubernetes cluster (see [below for nested schema](#nestedatt--exec))
- `host` (String) The address of the Kubernetes cluster (scheme://hostname:port)
- `insecure` (Boolean) Whether server should be accessed without verifying the TLS certificate
- `labels` (Attributes List) List of labels to apply to the runtime (see [below for nested schema](#nestedatt--labels))
- `password` (String) Password for basic authentication to the Kubernetes cluster
- `proxy_url` (String) Proxy URL to use when accessing the Kubernetes cluster
- `timeout` (String) How long to wait for the runtime linking to complete. A valid Go duration string, e.g. `10m` or `1h`. Defaults to `10m`
- `tls_server_name` (String) Server name passed to the server for SNI and is used in the client to check server certificates against
- `token` (String) Token to authenticate an service account
- `username` (String) Username for basic authentication to the Kubernetes cluster

### Read-Only

- `agent_externally_managed` (Boolean) If the agent has been set to be externally managed. This should be false since this is the managed_k8s_runtime resource -- this is used to detect out of band changes to the agent deployment
- `agent_namespace` (String) The namespace of the agent
- `agent_runtime_id` (String) The runtime identifier of the agent
- `id` (String) Runtime identifier

<a id="nestedatt--exec"></a>
### Nested Schema for `exec`

Required:

- `api_version` (String) API version of the exec credential plugin
- `command` (String) Command to execute

Optional:

- `args` (List of String) Arguments to pass when executing the command
- `env` (Map of String) Environment variables to set when executing the command


<a id="nestedatt--labels"></a>
### Nested Schema for `labels`

Required:

- `label` (String) Label name
- `value` (String) Label value
