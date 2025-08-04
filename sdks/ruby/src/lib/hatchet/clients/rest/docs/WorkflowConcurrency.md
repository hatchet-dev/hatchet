# HatchetSdkRest::WorkflowConcurrency

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **max_runs** | **Integer** | The maximum number of concurrent workflow runs. |  |
| **limit_strategy** | [**ConcurrencyLimitStrategy**](ConcurrencyLimitStrategy.md) | The strategy to use when the concurrency limit is reached. |  |
| **get_concurrency_group** | **String** | An action which gets the concurrency group for the WorkflowRun. |  |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::WorkflowConcurrency.new(
  max_runs: null,
  limit_strategy: null,
  get_concurrency_group: null
)
```

