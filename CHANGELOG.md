## 0.1.13 (Unreleased)

## 0.1.12

FEATURES:
  - Add runtime label support to `prodvana_k8s_runtime`

CHANGES:
  - Removed runtime label attribute from `prodvana_runtime_link` (labels should be set on `prodvana_k8s_runtime` or `prodvana_managed_k8s_runtime` instead)

## 0.1.11

FIXES:
  - To match API behavior, `prodvana_release_channel` `manual_approval_preconditions.name` is now optional

CHANGES:
  - `prodvana_release_channel.release_channel_stable_preconditions.duration` has been removed

## 0.1.10

FEATURES:
  - Add runtime label support to `prodvana_managed_k8s_runtime` and `prodvana_runtime_link`

## 0.1.9

FIXES:
  - Fixed a bug when setting the properties `exec.args` and `exec.env` on `prodvana_managed_k8s_runtime`

## 0.1.8

FIXES:
  - Added the `enabled` field to `prodvana_release_channel` protection lifecycle objects
    - This helps with CDK compatibility 


## 0.1.7

CHANGES:
  - Added an `enabled` field to `prodvana_release_channel` protection lifecycle objects
    - This helps with CDK compatibility 

## 0.1.6

FEATURES:
 - `prodvana_release_channel`
   - added ReleaseChannelStable and ManualApproval precondition support
   - Added Protections support
   - Added Constants support

## 0.1.5

FIXES:
 - Fixed a bug in `prodvana_managed_k8s_runtime` causing an error when attempting to apply after the first successful apply.
 - Fixed a bug in `prodvana_managed_k8s_runtime` around parsing authentication `exec` arguments.

## 0.1.4

FEATURES:
- Added `prodvana_runtime_link` resource
- Added `prodvana_managed_k8s_runtime` resource

CHANGES:
- `prodvana_runtime` resource removed and replaced with `prodvana_k8s_runtime`
- `prodvana_runtime` data source removed and replaced with `prodvana_k8s_runtime`

FIXES:
- Better handling when resources are deleted outside terraform

## 0.1.3

See 0.1.4, 0.1.3 was released prematurely.

## 0.1.2

BUGS:
- Fixed a bug where an incorrect runtime type would be set on Release Channel configuration

## 0.1.1

FEATURES:
- Added initial alpha support for `runtime` resources
- Added support for setting `k8s_namespace` and `ecs_prefix` on `prodvana_release_channel.runtime` definitions

BUGS:
- Fixed a bug in validation when setting the type of a Runtime attached to a Release Channel
