# HatchetSdkRest::APIMetaIntegration

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **name** | **String** | the name of the integration |  |
| **enabled** | **Boolean** | whether this integration is enabled on the instance |  |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::APIMetaIntegration.new(
  name: github,
  enabled: null
)
```

