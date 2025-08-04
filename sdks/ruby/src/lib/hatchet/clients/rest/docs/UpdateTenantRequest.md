# HatchetSdkRest::UpdateTenantRequest

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **name** | **String** | The name of the tenant. | [optional] |
| **analytics_opt_out** | **Boolean** | Whether the tenant has opted out of analytics. | [optional] |
| **alert_member_emails** | **Boolean** | Whether to alert tenant members. | [optional] |
| **enable_workflow_run_failure_alerts** | **Boolean** | Whether to send alerts when workflow runs fail. | [optional] |
| **enable_expiring_token_alerts** | **Boolean** | Whether to enable alerts when tokens are approaching expiration. | [optional] |
| **enable_tenant_resource_limit_alerts** | **Boolean** | Whether to enable alerts when tenant resources are approaching limits. | [optional] |
| **max_alerting_frequency** | **String** | The max frequency at which to alert. | [optional] |
| **version** | [**TenantVersion**](TenantVersion.md) | The version of the tenant. | [optional] |
| **ui_version** | [**TenantUIVersion**](TenantUIVersion.md) | The UI of the tenant. | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::UpdateTenantRequest.new(
  name: null,
  analytics_opt_out: null,
  alert_member_emails: null,
  enable_workflow_run_failure_alerts: null,
  enable_expiring_token_alerts: null,
  enable_tenant_resource_limit_alerts: null,
  max_alerting_frequency: null,
  version: null,
  ui_version: null
)
```

