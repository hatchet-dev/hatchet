# HatchetSdkRest::V1TriggerWorkflowRunRequest

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **workflow_name** | **String** | The name of the workflow. |  |
| **input** | **Object** |  |  |
| **additional_metadata** | **Object** |  | [optional] |
| **priority** | **Integer** | The priority of the workflow run. | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::V1TriggerWorkflowRunRequest.new(
  workflow_name: null,
  input: null,
  additional_metadata: null,
  priority: null
)
```

