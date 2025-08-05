# HatchetSdkRest::V1Filter

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **metadata** | [**APIResourceMeta**](APIResourceMeta.md) |  |  |
| **tenant_id** | **String** | The ID of the tenant associated with this filter. |  |
| **workflow_id** | **String** | The workflow id associated with this filter. |  |
| **scope** | **String** | The scope associated with this filter. Used for subsetting candidate filters at evaluation time |  |
| **expression** | **String** | The expression associated with this filter. |  |
| **payload** | **Object** | Additional payload data associated with the filter |  |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::V1Filter.new(
  metadata: null,
  tenant_id: null,
  workflow_id: null,
  scope: null,
  expression: null,
  payload: null
)
```

