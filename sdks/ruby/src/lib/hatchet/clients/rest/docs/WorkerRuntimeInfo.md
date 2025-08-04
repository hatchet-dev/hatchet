# HatchetSdkRest::WorkerRuntimeInfo

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **sdk_version** | **String** |  | [optional] |
| **language** | [**WorkerRuntimeSDKs**](WorkerRuntimeSDKs.md) |  | [optional] |
| **language_version** | **String** |  | [optional] |
| **os** | **String** |  | [optional] |
| **runtime_extra** | **String** |  | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::WorkerRuntimeInfo.new(
  sdk_version: null,
  language: null,
  language_version: null,
  os: null,
  runtime_extra: null
)
```

