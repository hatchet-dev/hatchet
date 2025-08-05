# HatchetSdkRest::TenantResourceLimit

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **metadata** | [**APIResourceMeta**](APIResourceMeta.md) |  |  |
| **resource** | [**TenantResource**](TenantResource.md) | The resource associated with this limit. |  |
| **limit_value** | **Integer** | The limit associated with this limit. |  |
| **value** | **Integer** | The current value associated with this limit. |  |
| **alarm_value** | **Integer** | The alarm value associated with this limit to warn of approaching limit value. | [optional] |
| **window** | **String** | The meter window for the limit. (i.e. 1 day, 1 week, 1 month) | [optional] |
| **last_refill** | **Time** | The last time the limit was refilled. | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::TenantResourceLimit.new(
  metadata: null,
  resource: null,
  limit_value: null,
  value: null,
  alarm_value: null,
  window: null,
  last_refill: null
)
```

