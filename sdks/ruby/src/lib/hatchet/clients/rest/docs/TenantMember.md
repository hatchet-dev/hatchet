# HatchetSdkRest::TenantMember

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **metadata** | [**APIResourceMeta**](APIResourceMeta.md) |  |  |
| **user** | [**UserTenantPublic**](UserTenantPublic.md) | The user associated with this tenant member. |  |
| **role** | [**TenantMemberRole**](TenantMemberRole.md) | The role of the user in the tenant. |  |
| **tenant** | [**Tenant**](Tenant.md) | The tenant associated with this tenant member. | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::TenantMember.new(
  metadata: null,
  user: null,
  role: null,
  tenant: null
)
```

