# HatchetSdkRest::APIMetaPosthog

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **api_key** | **String** | the PostHog API key | [optional] |
| **api_host** | **String** | the PostHog API host | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::APIMetaPosthog.new(
  api_key: phk_1234567890abcdef,
  api_host: https://posthog.example.com
)
```

