# HatchetSdkRest::Tenant

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **metadata** | [**APIResourceMeta**](APIResourceMeta.md) |  |  |
| **name** | **String** | The name of the tenant. |  |
| **slug** | **String** | The slug of the tenant. |  |
| **version** | [**TenantVersion**](TenantVersion.md) | The version of the tenant. |  |
| **analytics_opt_out** | **Boolean** | Whether the tenant has opted out of analytics. | [optional] |
| **alert_member_emails** | **Boolean** | Whether to alert tenant members. | [optional] |
| **ui_version** | [**TenantUIVersion**](TenantUIVersion.md) | The UI of the tenant. | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::Tenant.new(
  metadata: null,
  name: null,
  slug: null,
  version: null,
  analytics_opt_out: null,
  alert_member_emails: null,
  ui_version: null
)
```

