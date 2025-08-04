# HatchetSdkRest::PaginationResponse

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **current_page** | **Integer** | the current page | [optional] |
| **next_page** | **Integer** | the next page | [optional] |
| **num_pages** | **Integer** | the total number of pages for listing | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::PaginationResponse.new(
  current_page: 2,
  next_page: 3,
  num_pages: 10
)
```

