# HatchetSdkRest::WorkflowMetrics

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **group_key_runs_count** | **Integer** | The number of runs for a specific group key (passed via filter) | [optional] |
| **group_key_count** | **Integer** | The total number of concurrency group keys. | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::WorkflowMetrics.new(
  group_key_runs_count: null,
  group_key_count: null
)
```

