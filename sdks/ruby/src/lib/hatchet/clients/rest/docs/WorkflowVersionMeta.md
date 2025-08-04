# HatchetSdkRest::WorkflowVersionMeta

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **metadata** | [**APIResourceMeta**](APIResourceMeta.md) |  |  |
| **version** | **String** | The version of the workflow. |  |
| **order** | **Integer** |  |  |
| **workflow_id** | **String** |  |  |
| **workflow** | [**Workflow**](Workflow.md) |  | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::WorkflowVersionMeta.new(
  metadata: null,
  version: null,
  order: null,
  workflow_id: null,
  workflow: null
)
```

