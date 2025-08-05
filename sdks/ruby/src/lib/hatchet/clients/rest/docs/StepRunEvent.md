# HatchetSdkRest::StepRunEvent

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **id** | **Integer** |  |  |
| **time_first_seen** | **Time** |  |  |
| **time_last_seen** | **Time** |  |  |
| **reason** | [**StepRunEventReason**](StepRunEventReason.md) |  |  |
| **severity** | [**StepRunEventSeverity**](StepRunEventSeverity.md) |  |  |
| **message** | **String** |  |  |
| **count** | **Integer** |  |  |
| **step_run_id** | **String** |  | [optional] |
| **workflow_run_id** | **String** |  | [optional] |
| **data** | **Object** |  | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::StepRunEvent.new(
  id: null,
  time_first_seen: null,
  time_last_seen: null,
  reason: null,
  severity: null,
  message: null,
  count: null,
  step_run_id: null,
  workflow_run_id: null,
  data: null
)
```

