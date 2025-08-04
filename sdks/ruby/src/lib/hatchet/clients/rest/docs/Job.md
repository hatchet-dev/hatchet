# HatchetSdkRest::Job

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **metadata** | [**APIResourceMeta**](APIResourceMeta.md) |  |  |
| **tenant_id** | **String** |  |  |
| **version_id** | **String** |  |  |
| **name** | **String** |  |  |
| **steps** | [**Array&lt;Step&gt;**](Step.md) |  |  |
| **description** | **String** | The description of the job. | [optional] |
| **timeout** | **String** | The timeout of the job. | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::Job.new(
  metadata: null,
  tenant_id: null,
  version_id: null,
  name: null,
  steps: null,
  description: null,
  timeout: null
)
```

