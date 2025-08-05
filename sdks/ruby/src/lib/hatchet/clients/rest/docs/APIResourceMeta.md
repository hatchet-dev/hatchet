# HatchetSdkRest::APIResourceMeta

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **id** | **String** | the id of this resource, in UUID format |  |
| **created_at** | **Time** | the time that this resource was created |  |
| **updated_at** | **Time** | the time that this resource was last updated |  |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::APIResourceMeta.new(
  id: bb214807-246e-43a5-a25d-41761d1cff9e,
  created_at: 2022-12-13T15:06:48.888358-05:00,
  updated_at: 2022-12-13T15:06:48.888358-05:00
)
```

