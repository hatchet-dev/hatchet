# HatchetSdkRest::WorkflowRun

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **metadata** | [**APIResourceMeta**](APIResourceMeta.md) |  |  |
| **tenant_id** | **String** |  |  |
| **workflow_version_id** | **String** |  |  |
| **status** | [**WorkflowRunStatus**](WorkflowRunStatus.md) |  |  |
| **triggered_by** | [**WorkflowRunTriggeredBy**](WorkflowRunTriggeredBy.md) |  |  |
| **workflow_version** | [**WorkflowVersion**](WorkflowVersion.md) |  | [optional] |
| **display_name** | **String** |  | [optional] |
| **job_runs** | [**Array&lt;JobRun&gt;**](JobRun.md) |  | [optional] |
| **input** | **Hash&lt;String, Object&gt;** |  | [optional] |
| **error** | **String** |  | [optional] |
| **started_at** | **Time** |  | [optional] |
| **finished_at** | **Time** |  | [optional] |
| **duration** | **Integer** |  | [optional] |
| **parent_id** | **String** |  | [optional] |
| **parent_step_run_id** | **String** |  | [optional] |
| **additional_metadata** | **Hash&lt;String, Object&gt;** |  | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::WorkflowRun.new(
  metadata: null,
  tenant_id: null,
  workflow_version_id: null,
  status: null,
  triggered_by: null,
  workflow_version: null,
  display_name: null,
  job_runs: null,
  input: null,
  error: null,
  started_at: null,
  finished_at: null,
  duration: 1000,
  parent_id: bb214807-246e-43a5-a25d-41761d1cff9e,
  parent_step_run_id: bb214807-246e-43a5-a25d-41761d1cff9e,
  additional_metadata: null
)
```

