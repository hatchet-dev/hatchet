# HatchetSdkRest::V1LogLine

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **created_at** | **Time** | The creation date of the log line. |  |
| **message** | **String** | The log message. |  |
| **metadata** | **Object** | The log metadata. |  |
| **retry_count** | **Integer** | The retry count of the log line. | [optional] |
| **attempt** | **Integer** | The attempt number of the log line. | [optional] |
| **level** | [**V1LogLineLevel**](V1LogLineLevel.md) | The log level. | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::V1LogLine.new(
  created_at: null,
  message: null,
  metadata: null,
  retry_count: null,
  attempt: null,
  level: null
)
```

