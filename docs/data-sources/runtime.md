---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "prodvana_runtime Data Source - terraform-provider-prodvana"
subcategory: ""
description: |-
  Prodvana Runtime
---

# prodvana_runtime (Data Source)

Prodvana Runtime



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) Runtime name

### Optional

- `k8s` (Attributes) K8S Runtime Configuration Options. These are only valid when `type` is set to `K8S` (see [below for nested schema](#nestedatt--k8s))

### Read-Only

- `id` (String) Runtime identifier
- `type` (String) Type of the runtime, one of (ECS, EXTENSION, FAKE, K8S, PULUMI_RUNNER, TERRAFORM_RUNNER, UNKNOWN)

<a id="nestedatt--k8s"></a>
### Nested Schema for `k8s`

Optional:

- `agent_env` (Map of String) Environment variables to pass to the agent configuration. Useful for things like proxy configuration. Only useful when `agent_externally_managed` is false.
- `agent_externally_managed` (Boolean) Whether the agent lifecycle is handled externally by the runtime owner. When true, Prodvana will not update the agent. Default false.
- `api_token` (String, Sensitive) API Token used for linking the Kubernetes Prodvana agent

