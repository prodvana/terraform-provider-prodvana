---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "prodvana_runtime Resource - terraform-provider-prodvana"
subcategory: ""
description: |-
  (Alpha! This feature is still in progress.) This resource allows you to manage a Prodvana Runtime https://docs.prodvana.io/docs/prodvana-concepts#runtime.
---

# prodvana_runtime (Resource)

(Alpha! This feature is still in progress.) This resource allows you to manage a Prodvana [Runtime](https://docs.prodvana.io/docs/prodvana-concepts#runtime).

## Example Usage

```terraform
resource "prodvana_runtime" "example" {
  name = "my-runtime"
  type = "K8S"
  k8s = {
    agent_env = {
      "PROXY" = "example.com:8080"
    }
  }
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) Runtime name
- `type` (String) Type of the runtime, one of (K8S)

### Optional

- `k8s` (Attributes) K8S Runtime Configuration Options. These are only valid when `type` is set to `K8S` (see [below for nested schema](#nestedatt--k8s))

### Read-Only

- `id` (String) Runtime identifier

<a id="nestedatt--k8s"></a>
### Nested Schema for `k8s`

Optional:

- `agent_env` (Map of String) Environment variables to pass to the agent configuration. Useful for things like proxy configuration. Only useful when `agent_externally_managed` is false.
- `agent_externally_managed` (Boolean) Whether the agent lifecycle is handled externally by the runtime owner. When true, Prodvana will not update the agent. Default false.
- `api_token` (String, Sensitive) API Token used for linking the Kubernetes Prodvana agent

## Import

Import is supported using the following syntax:

```shell
$ terraform import prodvana_runtime.example <runtime name>
```