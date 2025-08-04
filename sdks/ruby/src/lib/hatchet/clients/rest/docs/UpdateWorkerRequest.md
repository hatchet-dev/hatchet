# HatchetSdkRest::UpdateWorkerRequest

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **is_paused** | **Boolean** | Whether the worker is paused and cannot accept new runs. | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::UpdateWorkerRequest.new(
  is_paused: null
)
```

