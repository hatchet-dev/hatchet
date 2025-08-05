# HatchetSdkRest::WorkflowList

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **metadata** | [**APIResourceMeta**](APIResourceMeta.md) |  | [optional] |
| **rows** | [**Array&lt;Workflow&gt;**](Workflow.md) |  | [optional] |
| **pagination** | [**PaginationResponse**](PaginationResponse.md) |  | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::WorkflowList.new(
  metadata: null,
  rows: null,
  pagination: null
)
```

