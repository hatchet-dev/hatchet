# HatchetSdkRest::StepRunArchive

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **step_run_id** | **String** |  |  |
| **order** | **Integer** |  |  |
| **retry_count** | **Integer** |  |  |
| **created_at** | **Time** |  |  |
| **input** | **String** |  | [optional] |
| **output** | **String** |  | [optional] |
| **started_at** | **Time** |  | [optional] |
| **error** | **String** |  | [optional] |
| **started_at_epoch** | **Integer** |  | [optional] |
| **finished_at** | **Time** |  | [optional] |
| **finished_at_epoch** | **Integer** |  | [optional] |
| **timeout_at** | **Time** |  | [optional] |
| **timeout_at_epoch** | **Integer** |  | [optional] |
| **cancelled_at** | **Time** |  | [optional] |
| **cancelled_at_epoch** | **Integer** |  | [optional] |
| **cancelled_reason** | **String** |  | [optional] |
| **cancelled_error** | **String** |  | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::StepRunArchive.new(
  step_run_id: null,
  order: null,
  retry_count: null,
  created_at: null,
  input: null,
  output: null,
  started_at: null,
  error: null,
  started_at_epoch: null,
  finished_at: null,
  finished_at_epoch: null,
  timeout_at: null,
  timeout_at_epoch: null,
  cancelled_at: null,
  cancelled_at_epoch: null,
  cancelled_reason: null,
  cancelled_error: null
)
```

