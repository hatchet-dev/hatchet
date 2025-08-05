# HatchetSdkRest::APIError

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **description** | **String** | a description for this error |  |
| **code** | **Integer** | a custom Hatchet error code | [optional] |
| **field** | **String** | the field that this error is associated with, if applicable | [optional] |
| **docs_link** | **String** | a link to the documentation for this error, if it exists | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::APIError.new(
  description: A descriptive error message,
  code: 1400,
  field: name,
  docs_link: github.com/hatchet-dev/hatchet
)
```

