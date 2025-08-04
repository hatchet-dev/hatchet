# HatchetSdkRest::WorkflowVersion

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **metadata** | [**APIResourceMeta**](APIResourceMeta.md) |  |  |
| **version** | **String** | The version of the workflow. |  |
| **order** | **Integer** |  |  |
| **workflow_id** | **String** |  |  |
| **sticky** | **String** | The sticky strategy of the workflow. | [optional] |
| **default_priority** | **Integer** | The default priority of the workflow. | [optional] |
| **workflow** | [**Workflow**](Workflow.md) |  | [optional] |
| **concurrency** | [**WorkflowConcurrency**](WorkflowConcurrency.md) |  | [optional] |
| **triggers** | [**WorkflowTriggers**](WorkflowTriggers.md) |  | [optional] |
| **schedule_timeout** | **String** |  | [optional] |
| **jobs** | [**Array&lt;Job&gt;**](Job.md) |  | [optional] |
| **workflow_config** | **Object** |  | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::WorkflowVersion.new(
  metadata: null,
  version: null,
  order: null,
  workflow_id: null,
  sticky: null,
  default_priority: null,
  workflow: null,
  concurrency: null,
  triggers: null,
  schedule_timeout: null,
  jobs: null,
  workflow_config: null
)
```

