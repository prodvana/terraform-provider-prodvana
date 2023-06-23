---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "prodvana_application Data Source - terraform-provider-prodvana"
subcategory: ""
description: |-
  Prodvana Application
---

# prodvana_application (Data Source)

Prodvana Application

## Example Usage

```terraform
data "prodvana_application" "example" {
  name = "my-app"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) Application name

### Read-Only

- `id` (String) Application identifier
- `version` (String) Current application version

