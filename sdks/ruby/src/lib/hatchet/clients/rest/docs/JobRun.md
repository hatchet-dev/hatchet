# HatchetSdkRest::JobRun

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **metadata** | [**APIResourceMeta**](APIResourceMeta.md) |  |  |
| **tenant_id** | **String** |  |  |
| **workflow_run_id** | **String** |  |  |
| **job_id** | **String** |  |  |
| **status** | [**JobRunStatus**](JobRunStatus.md) |  |  |
| **workflow_run** | [**WorkflowRun**](WorkflowRun.md) |  | [optional] |
| **job** | [**Job**](Job.md) |  | [optional] |
| **ticker_id** | **String** |  | [optional] |
| **step_runs** | [**Array&lt;StepRun&gt;**](StepRun.md) |  | [optional] |
| **result** | **Object** |  | [optional] |
| **started_at** | **Time** |  | [optional] |
| **finished_at** | **Time** |  | [optional] |
| **timeout_at** | **Time** |  | [optional] |
| **cancelled_at** | **Time** |  | [optional] |
| **cancelled_reason** | **String** |  | [optional] |
| **cancelled_error** | **String** |  | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::JobRun.new(
  metadata: null,
  tenant_id: null,
  workflow_run_id: null,
  job_id: null,
  status: null,
  workflow_run: null,
  job: null,
  ticker_id: null,
  step_runs: null,
  result: null,
  started_at: null,
  finished_at: null,
  timeout_at: null,
  cancelled_at: null,
  cancelled_reason: null,
  cancelled_error: null
)
```

