# HatchetSdkRest::V1TaskSummary

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **metadata** | [**APIResourceMeta**](APIResourceMeta.md) |  |  |
| **created_at** | **Time** | The timestamp the task was created. |  |
| **display_name** | **String** | The display name of the task run. |  |
| **input** | **Object** | The input of the task run. |  |
| **num_spawned_children** | **Integer** | The number of spawned children tasks |  |
| **output** | **Object** | The output of the task run (for the latest run) |  |
| **status** | [**V1TaskStatus**](V1TaskStatus.md) |  |  |
| **task_external_id** | **String** | The external ID of the task. |  |
| **task_id** | **Integer** | The ID of the task. |  |
| **task_inserted_at** | **Time** | The timestamp the task was inserted. |  |
| **tenant_id** | **String** | The ID of the tenant. |  |
| **type** | [**V1WorkflowType**](V1WorkflowType.md) | The type of the workflow (whether it&#39;s a DAG or a task) |  |
| **workflow_id** | **String** |  |  |
| **workflow_run_external_id** | **String** | The external ID of the workflow run |  |
| **action_id** | **String** | The action ID of the task. | [optional] |
| **retry_count** | **Integer** | The number of retries of the task. | [optional] |
| **attempt** | **Integer** | The attempt number of the task. | [optional] |
| **additional_metadata** | **Object** | Additional metadata for the task run. | [optional] |
| **children** | [**Array&lt;V1TaskSummary&gt;**](V1TaskSummary.md) | The list of children tasks | [optional] |
| **duration** | **Integer** | The duration of the task run, in milliseconds. | [optional] |
| **error_message** | **String** | The error message of the task run (for the latest run) | [optional] |
| **finished_at** | **Time** | The timestamp the task run finished. | [optional] |
| **started_at** | **Time** | The timestamp the task run started. | [optional] |
| **step_id** | **String** | The step ID of the task. | [optional] |
| **workflow_name** | **String** |  | [optional] |
| **workflow_version_id** | **String** | The version ID of the workflow | [optional] |
| **workflow_config** | **Object** |  | [optional] |
| **parent_task_external_id** | **String** | The external ID of the parent task. | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::V1TaskSummary.new(
  metadata: null,
  created_at: null,
  display_name: null,
  input: null,
  num_spawned_children: null,
  output: null,
  status: null,
  task_external_id: null,
  task_id: null,
  task_inserted_at: null,
  tenant_id: bb214807-246e-43a5-a25d-41761d1cff9e,
  type: null,
  workflow_id: null,
  workflow_run_external_id: null,
  action_id: null,
  retry_count: null,
  attempt: null,
  additional_metadata: null,
  children: null,
  duration: null,
  error_message: null,
  finished_at: null,
  started_at: null,
  step_id: null,
  workflow_name: null,
  workflow_version_id: null,
  workflow_config: null,
  parent_task_external_id: null
)
```

