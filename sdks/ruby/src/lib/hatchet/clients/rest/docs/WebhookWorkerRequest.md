# HatchetSdkRest::WebhookWorkerRequest

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **created_at** | **Time** | The date and time the request was created. |  |
| **method** | [**WebhookWorkerRequestMethod**](WebhookWorkerRequestMethod.md) | The HTTP method used for the request. |  |
| **status_code** | **Integer** | The HTTP status code of the response. |  |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::WebhookWorkerRequest.new(
  created_at: null,
  method: null,
  status_code: null
)
```

