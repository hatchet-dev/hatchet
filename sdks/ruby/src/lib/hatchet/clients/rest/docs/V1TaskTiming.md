# HatchetSdkRest::V1TaskTiming

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **metadata** | [**APIResourceMeta**](APIResourceMeta.md) |  |  |
| **depth** | **Integer** | The depth of the task in the waterfall. |  |
| **status** | [**V1TaskStatus**](V1TaskStatus.md) |  |  |
| **task_display_name** | **String** | The display name of the task run. |  |
| **task_external_id** | **String** | The external ID of the task. |  |
| **task_id** | **Integer** | The ID of the task. |  |
| **task_inserted_at** | **Time** | The timestamp the task was inserted. |  |
| **tenant_id** | **String** | The ID of the tenant. |  |
| **parent_task_external_id** | **String** | The external ID of the parent task. | [optional] |
| **queued_at** | **Time** | The timestamp the task run was queued. | [optional] |
| **started_at** | **Time** | The timestamp the task run started. | [optional] |
| **finished_at** | **Time** | The timestamp the task run finished. | [optional] |
| **workflow_run_id** | **String** | The external ID of the workflow run. | [optional] |
| **retry_count** | **Integer** | The number of retries of the task. | [optional] |
| **attempt** | **Integer** | The attempt number of the task. | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::V1TaskTiming.new(
  metadata: null,
  depth: null,
  status: null,
  task_display_name: null,
  task_external_id: null,
  task_id: null,
  task_inserted_at: null,
  tenant_id: bb214807-246e-43a5-a25d-41761d1cff9e,
  parent_task_external_id: null,
  queued_at: null,
  started_at: null,
  finished_at: null,
  workflow_run_id: null,
  retry_count: null,
  attempt: null
)
```

