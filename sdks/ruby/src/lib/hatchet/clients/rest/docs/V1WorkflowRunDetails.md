# HatchetSdkRest::V1WorkflowRunDetails

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **run** | [**V1WorkflowRun**](V1WorkflowRun.md) |  |  |
| **task_events** | [**Array&lt;V1TaskEvent&gt;**](V1TaskEvent.md) | The list of task events for the workflow run |  |
| **shape** | [**Array&lt;WorkflowRunShapeItemForWorkflowRunDetails&gt;**](WorkflowRunShapeItemForWorkflowRunDetails.md) |  |  |
| **tasks** | [**Array&lt;V1TaskSummary&gt;**](V1TaskSummary.md) |  |  |
| **workflow_config** | **Object** |  | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::V1WorkflowRunDetails.new(
  run: null,
  task_events: null,
  shape: null,
  tasks: null,
  workflow_config: null
)
```

