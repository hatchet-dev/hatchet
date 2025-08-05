# HatchetSdkRest::TenantInvite

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **metadata** | [**APIResourceMeta**](APIResourceMeta.md) |  |  |
| **email** | **String** | The email of the user to invite. |  |
| **role** | [**TenantMemberRole**](TenantMemberRole.md) | The role of the user in the tenant. |  |
| **tenant_id** | **String** | The tenant id associated with this tenant invite. |  |
| **expires** | **Time** | The time that this invite expires. |  |
| **tenant_name** | **String** | The tenant name for the tenant. | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::TenantInvite.new(
  metadata: null,
  email: null,
  role: null,
  tenant_id: null,
  expires: null,
  tenant_name: null
)
```

