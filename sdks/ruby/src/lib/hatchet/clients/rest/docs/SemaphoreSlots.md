# HatchetSdkRest::SemaphoreSlots

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **step_run_id** | **String** | The step run id. |  |
| **action_id** | **String** | The action id. |  |
| **workflow_run_id** | **String** | The workflow run id. |  |
| **started_at** | **Time** | The time this slot was started. | [optional] |
| **timeout_at** | **Time** | The time this slot will timeout. | [optional] |
| **status** | [**StepRunStatus**](StepRunStatus.md) |  | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::SemaphoreSlots.new(
  step_run_id: null,
  action_id: null,
  workflow_run_id: null,
  started_at: null,
  timeout_at: null,
  status: null
)
```

