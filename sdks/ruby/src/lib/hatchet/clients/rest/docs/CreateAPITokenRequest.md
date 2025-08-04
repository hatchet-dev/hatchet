# HatchetSdkRest::CreateAPITokenRequest

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **name** | **String** | A name for the API token. |  |
| **expires_in** | **String** | The duration for which the token is valid. | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::CreateAPITokenRequest.new(
  name: null,
  expires_in: null
)
```

