# HatchetSdkRest::TenantAlertingSettings

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **metadata** | [**APIResourceMeta**](APIResourceMeta.md) |  |  |
| **max_alerting_frequency** | **String** | The max frequency at which to alert. |  |
| **alert_member_emails** | **Boolean** | Whether to alert tenant members. | [optional] |
| **enable_workflow_run_failure_alerts** | **Boolean** | Whether to send alerts when workflow runs fail. | [optional] |
| **enable_expiring_token_alerts** | **Boolean** | Whether to enable alerts when tokens are approaching expiration. | [optional] |
| **enable_tenant_resource_limit_alerts** | **Boolean** | Whether to enable alerts when tenant resources are approaching limits. | [optional] |
| **last_alerted_at** | **Time** | The last time an alert was sent. | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::TenantAlertingSettings.new(
  metadata: null,
  max_alerting_frequency: null,
  alert_member_emails: null,
  enable_workflow_run_failure_alerts: null,
  enable_expiring_token_alerts: null,
  enable_tenant_resource_limit_alerts: null,
  last_alerted_at: null
)
```

