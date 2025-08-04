# HatchetSdkRest::V1ReplayTaskRequest

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **external_ids** | **Array&lt;String&gt;** | A list of external IDs, which can refer to either task or workflow run external IDs | [optional] |
| **filter** | [**V1TaskFilter**](V1TaskFilter.md) |  | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::V1ReplayTaskRequest.new(
  external_ids: null,
  filter: null
)
```

