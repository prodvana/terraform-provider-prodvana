## 0.1.3 (Unreleased)

FEATURES:
- Added `prodvana_runtime_link` resource
- Added `prodvana_managed_k8s_runtime` resource

CHANGES:
- `prodvana_runtime` resource removed and replaced with `prodvana_k8s_runtime`
- `prodvana_runtime` data source removed and replaced with `prodvana_k8s_runtime`

FIXES:
- Better handling when resources are deleted outside terraform

## 0.1.2

BUGS:
- Fixed a bug where an incorrect runtime type would be set on Release Channel configuration

## 0.1.1

FEATURES:
- Added initial alpha support for `runtime` resources
- Added support for setting `k8s_namespace` and `ecs_prefix` on `prodvana_release_channel.runtime` definitions

BUGS:
- Fixed a bug in validation when setting the type of a Runtime attached to a Release Channel
