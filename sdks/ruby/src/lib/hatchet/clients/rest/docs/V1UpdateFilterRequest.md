# HatchetSdkRest::V1UpdateFilterRequest

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **expression** | **String** | The expression for the filter | [optional] |
| **scope** | **String** | The scope associated with this filter. Used for subsetting candidate filters at evaluation time | [optional] |
| **payload** | **Object** | The payload for the filter | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::V1UpdateFilterRequest.new(
  expression: null,
  scope: null,
  payload: null
)
```

