# HatchetSdkRest::WorkflowRunTriggeredBy

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **metadata** | [**APIResourceMeta**](APIResourceMeta.md) |  |  |
| **parent_workflow_run_id** | **String** |  | [optional] |
| **event_id** | **String** |  | [optional] |
| **cron_parent_id** | **String** |  | [optional] |
| **cron_schedule** | **String** |  | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::WorkflowRunTriggeredBy.new(
  metadata: null,
  parent_workflow_run_id: null,
  event_id: null,
  cron_parent_id: null,
  cron_schedule: null
)
```

