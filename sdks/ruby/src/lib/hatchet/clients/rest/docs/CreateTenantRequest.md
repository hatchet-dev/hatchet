# HatchetSdkRest::CreateTenantRequest

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **name** | **String** | The name of the tenant. |  |
| **slug** | **String** | The slug of the tenant. |  |
| **ui_version** | [**TenantUIVersion**](TenantUIVersion.md) | The UI version of the tenant. Defaults to V0. | [optional] |
| **engine_version** | [**TenantVersion**](TenantVersion.md) | The engine version of the tenant. Defaults to V0. | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::CreateTenantRequest.new(
  name: null,
  slug: null,
  ui_version: null,
  engine_version: null
)
```

