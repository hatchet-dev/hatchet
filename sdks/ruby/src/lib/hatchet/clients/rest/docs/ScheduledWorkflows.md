# HatchetSdkRest::ScheduledWorkflows

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **metadata** | [**APIResourceMeta**](APIResourceMeta.md) |  |  |
| **tenant_id** | **String** |  |  |
| **workflow_version_id** | **String** |  |  |
| **workflow_id** | **String** |  |  |
| **workflow_name** | **String** |  |  |
| **trigger_at** | **Time** |  |  |
| **method** | [**ScheduledWorkflowsMethod**](ScheduledWorkflowsMethod.md) |  |  |
| **input** | **Hash&lt;String, Object&gt;** |  | [optional] |
| **additional_metadata** | **Hash&lt;String, Object&gt;** |  | [optional] |
| **workflow_run_created_at** | **Time** |  | [optional] |
| **workflow_run_name** | **String** |  | [optional] |
| **workflow_run_status** | [**WorkflowRunStatus**](WorkflowRunStatus.md) |  | [optional] |
| **workflow_run_id** | **String** |  | [optional] |
| **priority** | **Integer** |  | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::ScheduledWorkflows.new(
  metadata: null,
  tenant_id: null,
  workflow_version_id: null,
  workflow_id: null,
  workflow_name: null,
  trigger_at: null,
  method: null,
  input: null,
  additional_metadata: null,
  workflow_run_created_at: null,
  workflow_run_name: null,
  workflow_run_status: null,
  workflow_run_id: bb214807-246e-43a5-a25d-41761d1cff9e,
  priority: null
)
```

