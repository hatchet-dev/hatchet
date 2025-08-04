# HatchetSdkRest::Event

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **metadata** | [**APIResourceMeta**](APIResourceMeta.md) |  |  |
| **key** | **String** | The key for the event. |  |
| **tenant_id** | **String** | The ID of the tenant associated with this event. |  |
| **tenant** | [**Tenant**](Tenant.md) | The tenant associated with this event. | [optional] |
| **workflow_run_summary** | [**EventWorkflowRunSummary**](EventWorkflowRunSummary.md) | The workflow run summary for this event. | [optional] |
| **additional_metadata** | **Object** | Additional metadata for the event. | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::Event.new(
  metadata: null,
  key: null,
  tenant_id: null,
  tenant: null,
  workflow_run_summary: null,
  additional_metadata: null
)
```

