# HatchetSdkRest::V1CreateFilterRequest

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **workflow_id** | **String** | The workflow id |  |
| **expression** | **String** | The expression for the filter |  |
| **scope** | **String** | The scope associated with this filter. Used for subsetting candidate filters at evaluation time |  |
| **payload** | **Object** | The payload for the filter | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::V1CreateFilterRequest.new(
  workflow_id: null,
  expression: null,
  scope: null,
  payload: null
)
```

