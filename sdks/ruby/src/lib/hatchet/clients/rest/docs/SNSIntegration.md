# HatchetSdkRest::SNSIntegration

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **metadata** | [**APIResourceMeta**](APIResourceMeta.md) |  |  |
| **tenant_id** | **String** | The unique identifier for the tenant that the SNS integration belongs to. |  |
| **topic_arn** | **String** | The Amazon Resource Name (ARN) of the SNS topic. |  |
| **ingest_url** | **String** | The URL to send SNS messages to. | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::SNSIntegration.new(
  metadata: null,
  tenant_id: null,
  topic_arn: null,
  ingest_url: null
)
```

