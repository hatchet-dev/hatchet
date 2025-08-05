# HatchetSdkRest::Workflow

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **metadata** | [**APIResourceMeta**](APIResourceMeta.md) |  |  |
| **name** | **String** | The name of the workflow. |  |
| **tenant_id** | **String** | The tenant id of the workflow. |  |
| **description** | **String** | The description of the workflow. | [optional] |
| **is_paused** | **Boolean** | Whether the workflow is paused. | [optional] |
| **versions** | [**Array&lt;WorkflowVersionMeta&gt;**](WorkflowVersionMeta.md) |  | [optional] |
| **tags** | [**Array&lt;WorkflowTag&gt;**](WorkflowTag.md) | The tags of the workflow. | [optional] |
| **jobs** | [**Array&lt;Job&gt;**](Job.md) | The jobs of the workflow. | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::Workflow.new(
  metadata: null,
  name: null,
  tenant_id: null,
  description: null,
  is_paused: null,
  versions: null,
  tags: null,
  jobs: null
)
```

