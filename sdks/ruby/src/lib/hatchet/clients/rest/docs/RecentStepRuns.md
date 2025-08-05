# HatchetSdkRest::RecentStepRuns

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **metadata** | [**APIResourceMeta**](APIResourceMeta.md) |  |  |
| **action_id** | **String** | The action id. |  |
| **status** | [**StepRunStatus**](StepRunStatus.md) |  |  |
| **workflow_run_id** | **String** |  |  |
| **started_at** | **Time** |  | [optional] |
| **finished_at** | **Time** |  | [optional] |
| **cancelled_at** | **Time** |  | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::RecentStepRuns.new(
  metadata: null,
  action_id: null,
  status: null,
  workflow_run_id: null,
  started_at: null,
  finished_at: null,
  cancelled_at: null
)
```

