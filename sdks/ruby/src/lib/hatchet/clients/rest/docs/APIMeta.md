# HatchetSdkRest::APIMeta

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **auth** | [**APIMetaAuth**](APIMetaAuth.md) |  | [optional] |
| **pylon_app_id** | **String** | the Pylon app ID for usepylon.com chat support | [optional] |
| **posthog** | [**APIMetaPosthog**](APIMetaPosthog.md) |  | [optional] |
| **allow_signup** | **Boolean** | whether or not users can sign up for this instance | [optional] |
| **allow_invites** | **Boolean** | whether or not users can invite other users to this instance | [optional] |
| **allow_create_tenant** | **Boolean** | whether or not users can create new tenants | [optional] |
| **allow_change_password** | **Boolean** | whether or not users can change their password | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::APIMeta.new(
  auth: null,
  pylon_app_id: 12345678-1234-1234-1234-123456789012,
  posthog: null,
  allow_signup: true,
  allow_invites: true,
  allow_create_tenant: true,
  allow_change_password: true
)
```

