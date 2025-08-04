# HatchetSdkRest::V1Event

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **metadata** | [**APIResourceMeta**](APIResourceMeta.md) |  |  |
| **key** | **String** | The key for the event. |  |
| **tenant_id** | **String** | The ID of the tenant associated with this event. |  |
| **workflow_run_summary** | [**V1EventWorkflowRunSummary**](V1EventWorkflowRunSummary.md) | The workflow run summary for this event. |  |
| **tenant** | [**Tenant**](Tenant.md) | The tenant associated with this event. | [optional] |
| **additional_metadata** | **Object** | Additional metadata for the event. | [optional] |
| **payload** | **Object** | The payload of the event, which can be any JSON-serializable object. | [optional] |
| **scope** | **String** | The scope of the event, which can be used to filter or categorize events. | [optional] |
| **seen_at** | **Time** | The timestamp when the event was seen. | [optional] |
| **triggered_runs** | [**Array&lt;V1EventTriggeredRun&gt;**](V1EventTriggeredRun.md) | The external IDs of the runs that were triggered by this event. | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::V1Event.new(
  metadata: null,
  key: null,
  tenant_id: null,
  workflow_run_summary: null,
  tenant: null,
  additional_metadata: null,
  payload: null,
  scope: null,
  seen_at: null,
  triggered_runs: null
)
```

