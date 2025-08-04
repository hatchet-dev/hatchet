# HatchetSdkRest::TenantQueueMetrics

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **total** | [**QueueMetrics**](QueueMetrics.md) | The total queue metrics. | [optional] |
| **workflow** | [**Hash&lt;String, QueueMetrics&gt;**](QueueMetrics.md) |  | [optional] |
| **queues** | **Hash&lt;String, Integer&gt;** |  | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::TenantQueueMetrics.new(
  total: null,
  workflow: null,
  queues: null
)
```

