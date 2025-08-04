# HatchetSdkRest::StepRun

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **metadata** | [**APIResourceMeta**](APIResourceMeta.md) |  |  |
| **tenant_id** | **String** |  |  |
| **job_run_id** | **String** |  |  |
| **step_id** | **String** |  |  |
| **status** | [**StepRunStatus**](StepRunStatus.md) |  |  |
| **job_run** | [**JobRun**](JobRun.md) |  | [optional] |
| **step** | [**Step**](Step.md) |  | [optional] |
| **child_workflows_count** | **Integer** |  | [optional] |
| **parents** | **Array&lt;String&gt;** |  | [optional] |
| **child_workflow_runs** | **Array&lt;String&gt;** |  | [optional] |
| **worker_id** | **String** |  | [optional] |
| **input** | **String** |  | [optional] |
| **output** | **String** |  | [optional] |
| **requeue_after** | **Time** |  | [optional] |
| **result** | **Object** |  | [optional] |
| **error** | **String** |  | [optional] |
| **started_at** | **Time** |  | [optional] |
| **started_at_epoch** | **Integer** |  | [optional] |
| **finished_at** | **Time** |  | [optional] |
| **finished_at_epoch** | **Integer** |  | [optional] |
| **timeout_at** | **Time** |  | [optional] |
| **timeout_at_epoch** | **Integer** |  | [optional] |
| **cancelled_at** | **Time** |  | [optional] |
| **cancelled_at_epoch** | **Integer** |  | [optional] |
| **cancelled_reason** | **String** |  | [optional] |
| **cancelled_error** | **String** |  | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::StepRun.new(
  metadata: null,
  tenant_id: null,
  job_run_id: null,
  step_id: null,
  status: null,
  job_run: null,
  step: null,
  child_workflows_count: null,
  parents: null,
  child_workflow_runs: null,
  worker_id: null,
  input: null,
  output: null,
  requeue_after: null,
  result: null,
  error: null,
  started_at: null,
  started_at_epoch: null,
  finished_at: null,
  finished_at_epoch: null,
  timeout_at: null,
  timeout_at_epoch: null,
  cancelled_at: null,
  cancelled_at_epoch: null,
  cancelled_reason: null,
  cancelled_error: null
)
```

