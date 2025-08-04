# HatchetSdkRest::EventWorkflowRunSummary

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **pending** | **Integer** | The number of pending runs. | [optional] |
| **running** | **Integer** | The number of running runs. | [optional] |
| **queued** | **Integer** | The number of queued runs. | [optional] |
| **succeeded** | **Integer** | The number of succeeded runs. | [optional] |
| **failed** | **Integer** | The number of failed runs. | [optional] |
| **cancelled** | **Integer** | The number of cancelled runs. | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::EventWorkflowRunSummary.new(
  pending: null,
  running: null,
  queued: null,
  succeeded: null,
  failed: null,
  cancelled: null
)
```

