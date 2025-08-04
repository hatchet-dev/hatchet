# HatchetSdkRest::CreateCronWorkflowTriggerRequest

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **input** | **Object** |  |  |
| **additional_metadata** | **Object** |  |  |
| **cron_name** | **String** |  |  |
| **cron_expression** | **String** |  |  |
| **priority** | **Integer** |  | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::CreateCronWorkflowTriggerRequest.new(
  input: null,
  additional_metadata: null,
  cron_name: null,
  cron_expression: null,
  priority: null
)
```

