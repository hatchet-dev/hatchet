# HatchetSdkRest::Step

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **metadata** | [**APIResourceMeta**](APIResourceMeta.md) |  |  |
| **readable_id** | **String** | The readable id of the step. |  |
| **tenant_id** | **String** |  |  |
| **job_id** | **String** |  |  |
| **action** | **String** |  |  |
| **timeout** | **String** | The timeout of the step. | [optional] |
| **children** | **Array&lt;String&gt;** |  | [optional] |
| **parents** | **Array&lt;String&gt;** |  | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::Step.new(
  metadata: null,
  readable_id: null,
  tenant_id: null,
  job_id: null,
  action: null,
  timeout: null,
  children: null,
  parents: null
)
```

