# HatchetSdkRest::V1TaskFilter

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **since** | **Time** |  |  |
| **_until** | **Time** |  | [optional] |
| **statuses** | [**Array&lt;V1TaskStatus&gt;**](V1TaskStatus.md) |  | [optional] |
| **workflow_ids** | **Array&lt;String&gt;** |  | [optional] |
| **additional_metadata** | **Array&lt;String&gt;** |  | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::V1TaskFilter.new(
  since: null,
  _until: null,
  statuses: null,
  workflow_ids: null,
  additional_metadata: null
)
```

