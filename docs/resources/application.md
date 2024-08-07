---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "prodvana_application Resource - terraform-provider-prodvana"
subcategory: ""
description: |-
  This resource allows you to manage a Prodvana Application https://docs.prodvana.io/docs/prodvana-concepts#application.
---

# prodvana_application (Resource)

This resource allows you to manage a Prodvana [Application](https://docs.prodvana.io/docs/prodvana-concepts#application).

## Example Usage

```terraform
resource "prodvana_application" "example" {
  name = "my-app"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) Application name

### Optional

- `description` (String) Application description
- `no_cleanup_on_delete` (Boolean) Prevent the application from being deleted when the resource is destroyed

### Read-Only

- `id` (String) Application identifier
- `version` (String) Current application version

## Import

Import is supported using the following syntax:

```shell
$ terraform import prodvana_application.example <application name>
```
