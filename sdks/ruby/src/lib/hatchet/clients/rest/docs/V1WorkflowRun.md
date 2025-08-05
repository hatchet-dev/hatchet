# HatchetSdkRest::V1WorkflowRun

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **metadata** | [**APIResourceMeta**](APIResourceMeta.md) |  |  |
| **status** | [**V1TaskStatus**](V1TaskStatus.md) |  |  |
| **tenant_id** | **String** | The ID of the tenant. |  |
| **display_name** | **String** | The display name of the task run. |  |
| **workflow_id** | **String** |  |  |
| **output** | **Object** | The output of the task run (for the latest run) |  |
| **input** | **Object** | The input of the task run. |  |
| **started_at** | **Time** | The timestamp the task run started. | [optional] |
| **finished_at** | **Time** | The timestamp the task run finished. | [optional] |
| **duration** | **Integer** | The duration of the task run, in milliseconds. | [optional] |
| **additional_metadata** | **Object** | Additional metadata for the task run. | [optional] |
| **error_message** | **String** | The error message of the task run (for the latest run) | [optional] |
| **workflow_version_id** | **String** | The ID of the workflow version. | [optional] |
| **created_at** | **Time** | The timestamp the task run was created. | [optional] |
| **parent_task_external_id** | **String** |  | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::V1WorkflowRun.new(
  metadata: null,
  status: null,
  tenant_id: bb214807-246e-43a5-a25d-41761d1cff9e,
  display_name: null,
  workflow_id: null,
  output: null,
  input: null,
  started_at: null,
  finished_at: null,
  duration: null,
  additional_metadata: null,
  error_message: null,
  workflow_version_id: null,
  created_at: null,
  parent_task_external_id: null
)
```

