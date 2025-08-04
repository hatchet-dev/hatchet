# HatchetSdkRest::RateLimit

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **key** | **String** | The key for the rate limit. |  |
| **tenant_id** | **String** | The ID of the tenant associated with this rate limit. |  |
| **limit_value** | **Integer** | The maximum number of requests allowed within the window. |  |
| **value** | **Integer** | The current number of requests made within the window. |  |
| **window** | **String** | The window of time in which the limitValue is enforced. |  |
| **last_refill** | **Time** | The last time the rate limit was refilled. |  |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::RateLimit.new(
  key: null,
  tenant_id: null,
  limit_value: null,
  value: null,
  window: null,
  last_refill: 2022-12-13T15:06:48.888358-05:00
)
```

