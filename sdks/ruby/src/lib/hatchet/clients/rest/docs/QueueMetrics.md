# HatchetSdkRest::QueueMetrics

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **num_queued** | **Integer** | The number of items in the queue. |  |
| **num_running** | **Integer** | The number of items running. |  |
| **num_pending** | **Integer** | The number of items pending. |  |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::QueueMetrics.new(
  num_queued: null,
  num_running: null,
  num_pending: null
)
```

