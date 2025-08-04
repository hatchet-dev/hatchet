# HatchetSdkRest::V1TaskEvent

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **id** | **Integer** |  |  |
| **task_id** | **String** |  |  |
| **timestamp** | **Time** |  |  |
| **event_type** | [**V1TaskEventType**](V1TaskEventType.md) |  |  |
| **message** | **String** |  |  |
| **error_message** | **String** |  | [optional] |
| **output** | **String** |  | [optional] |
| **worker_id** | **String** |  | [optional] |
| **task_display_name** | **String** |  | [optional] |
| **retry_count** | **Integer** | The number of retries of the task. | [optional] |
| **attempt** | **Integer** | The attempt number of the task. | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::V1TaskEvent.new(
  id: null,
  task_id: null,
  timestamp: null,
  event_type: null,
  message: null,
  error_message: null,
  output: null,
  worker_id: null,
  task_display_name: null,
  retry_count: null,
  attempt: null
)
```

