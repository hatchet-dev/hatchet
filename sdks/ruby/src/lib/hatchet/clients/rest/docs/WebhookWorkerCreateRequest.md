# HatchetSdkRest::WebhookWorkerCreateRequest

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **name** | **String** | The name of the webhook worker. |  |
| **url** | **String** | The webhook url. |  |
| **secret** | **String** | The secret key for validation. If not provided, a random secret will be generated. | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::WebhookWorkerCreateRequest.new(
  name: null,
  url: null,
  secret: null
)
```

