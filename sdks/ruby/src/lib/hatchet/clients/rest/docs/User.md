# HatchetSdkRest::User

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **metadata** | [**APIResourceMeta**](APIResourceMeta.md) |  |  |
| **email** | **String** | The email address of the user. |  |
| **email_verified** | **Boolean** | Whether the user has verified their email address. |  |
| **name** | **String** | The display name of the user. | [optional] |
| **has_password** | **Boolean** | Whether the user has a password set. | [optional] |
| **email_hash** | **String** | A hash of the user&#39;s email address for use with Pylon Support Chat | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::User.new(
  metadata: null,
  email: null,
  email_verified: null,
  name: null,
  has_password: null,
  email_hash: null
)
```

