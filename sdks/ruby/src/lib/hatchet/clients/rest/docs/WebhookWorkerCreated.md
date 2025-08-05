# HatchetSdkRest::WebhookWorkerCreated

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **metadata** | [**APIResourceMeta**](APIResourceMeta.md) |  |  |
| **name** | **String** | The name of the webhook worker. |  |
| **url** | **String** | The webhook url. |  |
| **secret** | **String** | The secret key for validation. |  |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::WebhookWorkerCreated.new(
  metadata: null,
  name: null,
  url: null,
  secret: null
)
```

