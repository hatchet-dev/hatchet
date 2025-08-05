# HatchetSdkRest::V1EventTriggeredRun

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **workflow_run_id** | **String** | The external ID of the triggered run. |  |
| **filter_id** | **String** | The ID of the filter that triggered the run, if applicable. | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::V1EventTriggeredRun.new(
  workflow_run_id: null,
  filter_id: null
)
```

