# HatchetSdkRest::APIToken

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **metadata** | [**APIResourceMeta**](APIResourceMeta.md) |  |  |
| **name** | **String** | The name of the API token. |  |
| **expires_at** | **Time** | When the API token expires. |  |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::APIToken.new(
  metadata: null,
  name: null,
  expires_at: null
)
```

