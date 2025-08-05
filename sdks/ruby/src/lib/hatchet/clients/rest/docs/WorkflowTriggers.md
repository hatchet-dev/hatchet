# HatchetSdkRest::WorkflowTriggers

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **metadata** | [**APIResourceMeta**](APIResourceMeta.md) |  | [optional] |
| **workflow_version_id** | **String** |  | [optional] |
| **tenant_id** | **String** |  | [optional] |
| **events** | [**Array&lt;WorkflowTriggerEventRef&gt;**](WorkflowTriggerEventRef.md) |  | [optional] |
| **crons** | [**Array&lt;WorkflowTriggerCronRef&gt;**](WorkflowTriggerCronRef.md) |  | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::WorkflowTriggers.new(
  metadata: null,
  workflow_version_id: null,
  tenant_id: null,
  events: null,
  crons: null
)
```

