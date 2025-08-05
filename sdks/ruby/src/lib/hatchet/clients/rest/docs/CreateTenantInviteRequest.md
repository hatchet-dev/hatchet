# HatchetSdkRest::CreateTenantInviteRequest

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **email** | **String** | The email of the user to invite. |  |
| **role** | [**TenantMemberRole**](TenantMemberRole.md) | The role of the user in the tenant. |  |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::CreateTenantInviteRequest.new(
  email: null,
  role: null
)
```

