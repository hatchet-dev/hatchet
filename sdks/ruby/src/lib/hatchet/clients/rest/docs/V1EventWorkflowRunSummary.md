# HatchetSdkRest::V1EventWorkflowRunSummary

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **running** | **Integer** | The number of running runs. |  |
| **queued** | **Integer** | The number of queued runs. |  |
| **succeeded** | **Integer** | The number of succeeded runs. |  |
| **failed** | **Integer** | The number of failed runs. |  |
| **cancelled** | **Integer** | The number of cancelled runs. |  |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::V1EventWorkflowRunSummary.new(
  running: null,
  queued: null,
  succeeded: null,
  failed: null,
  cancelled: null
)
```

