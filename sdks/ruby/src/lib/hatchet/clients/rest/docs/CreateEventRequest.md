# HatchetSdkRest::CreateEventRequest

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **key** | **String** | The key for the event. |  |
| **data** | **Object** | The data for the event. |  |
| **additional_metadata** | **Object** | Additional metadata for the event. | [optional] |
| **priority** | **Integer** | The priority of the event. | [optional] |
| **scope** | **String** | The scope for event filtering. | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::CreateEventRequest.new(
  key: null,
  data: null,
  additional_metadata: null,
  priority: null,
  scope: null
)
```

