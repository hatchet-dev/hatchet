# HatchetSdkRest::PullRequest

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **repository_owner** | **String** |  |  |
| **repository_name** | **String** |  |  |
| **pull_request_id** | **Integer** |  |  |
| **pull_request_title** | **String** |  |  |
| **pull_request_number** | **Integer** |  |  |
| **pull_request_head_branch** | **String** |  |  |
| **pull_request_base_branch** | **String** |  |  |
| **pull_request_state** | [**PullRequestState**](PullRequestState.md) |  |  |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::PullRequest.new(
  repository_owner: null,
  repository_name: null,
  pull_request_id: null,
  pull_request_title: null,
  pull_request_number: null,
  pull_request_head_branch: null,
  pull_request_base_branch: null,
  pull_request_state: null
)
```

