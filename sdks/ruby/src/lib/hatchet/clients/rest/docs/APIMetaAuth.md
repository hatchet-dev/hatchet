# HatchetSdkRest::APIMetaAuth

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **schemes** | **Array&lt;String&gt;** | the supported types of authentication | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::APIMetaAuth.new(
  schemes: [basic, google]
)
```

