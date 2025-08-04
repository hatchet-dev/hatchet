# HatchetSdkRest::SlackWebhook

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **metadata** | [**APIResourceMeta**](APIResourceMeta.md) |  |  |
| **tenant_id** | **String** | The unique identifier for the tenant that the SNS integration belongs to. |  |
| **team_name** | **String** | The team name associated with this slack webhook. |  |
| **team_id** | **String** | The team id associated with this slack webhook. |  |
| **channel_name** | **String** | The channel name associated with this slack webhook. |  |
| **channel_id** | **String** | The channel id associated with this slack webhook. |  |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::SlackWebhook.new(
  metadata: null,
  tenant_id: null,
  team_name: null,
  team_id: null,
  channel_name: null,
  channel_id: null
)
```

