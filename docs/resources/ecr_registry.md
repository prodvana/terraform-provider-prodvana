---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "prodvana_ecr_registry Resource - terraform-provider-prodvana"
subcategory: ""
description: |-
  This resource allows you to link an ECR registry https://docs.prodvana.io/docs/ecr to Prodvana.
---

# prodvana_ecr_registry (Resource)

This resource allows you to link an [ECR registry](https://docs.prodvana.io/docs/ecr) to Prodvana.



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `credentials_auth` (Attributes) Credentials to authenticate with the ECR registry. (see [below for nested schema](#nestedatt--credentials_auth))
- `name` (String) Name for the ECR registry, used to reference it in Prodvana configuration.
- `region` (String) AWS region where the ECR registry is located.

### Read-Only

- `id` (String) ECR Registry Identifier

<a id="nestedatt--credentials_auth"></a>
### Nested Schema for `credentials_auth`

Required:

- `access_key_id` (String) AWS Access Key ID with permissions to the ECR registry
- `secret_access_key` (String, Sensitive) AWS Secret Access Key with permissions to the ECR registry

