# HatchetSdkRest::TenantApi

All URIs are relative to *http://localhost*

| Method | HTTP request | Description |
| ------ | ------------ | ----------- |
| [**alert_email_group_create**](TenantApi.md#alert_email_group_create) | **POST** /api/v1/tenants/{tenant}/alerting-email-groups | Create tenant alert email group |
| [**alert_email_group_delete**](TenantApi.md#alert_email_group_delete) | **DELETE** /api/v1/alerting-email-groups/{alert-email-group} | Delete tenant alert email group |
| [**alert_email_group_list**](TenantApi.md#alert_email_group_list) | **GET** /api/v1/tenants/{tenant}/alerting-email-groups | List tenant alert email groups |
| [**alert_email_group_update**](TenantApi.md#alert_email_group_update) | **PATCH** /api/v1/alerting-email-groups/{alert-email-group} | Update tenant alert email group |
| [**tenant_alerting_settings_get**](TenantApi.md#tenant_alerting_settings_get) | **GET** /api/v1/tenants/{tenant}/alerting/settings | Get tenant alerting settings |
| [**tenant_create**](TenantApi.md#tenant_create) | **POST** /api/v1/tenants | Create tenant |
| [**tenant_get**](TenantApi.md#tenant_get) | **GET** /api/v1/tenants/{tenant} | Get tenant |
| [**tenant_get_prometheus_metrics**](TenantApi.md#tenant_get_prometheus_metrics) | **GET** /api/v1/tenants/{tenant}/prometheus-metrics | Get prometheus metrics |
| [**tenant_get_step_run_queue_metrics**](TenantApi.md#tenant_get_step_run_queue_metrics) | **GET** /api/v1/tenants/{tenant}/step-run-queue-metrics | Get step run metrics |
| [**tenant_invite_accept**](TenantApi.md#tenant_invite_accept) | **POST** /api/v1/users/invites/accept | Accept tenant invite |
| [**tenant_invite_create**](TenantApi.md#tenant_invite_create) | **POST** /api/v1/tenants/{tenant}/invites | Create tenant invite |
| [**tenant_invite_list**](TenantApi.md#tenant_invite_list) | **GET** /api/v1/tenants/{tenant}/invites | List tenant invites |
| [**tenant_invite_reject**](TenantApi.md#tenant_invite_reject) | **POST** /api/v1/users/invites/reject | Reject tenant invite |
| [**tenant_member_delete**](TenantApi.md#tenant_member_delete) | **DELETE** /api/v1/tenants/{tenant}/members/{member} | Delete a tenant member |
| [**tenant_member_list**](TenantApi.md#tenant_member_list) | **GET** /api/v1/tenants/{tenant}/members | List tenant members |
| [**tenant_resource_policy_get**](TenantApi.md#tenant_resource_policy_get) | **GET** /api/v1/tenants/{tenant}/resource-policy | Create tenant alert email group |
| [**tenant_update**](TenantApi.md#tenant_update) | **PATCH** /api/v1/tenants/{tenant} | Update tenant |
| [**user_list_tenant_invites**](TenantApi.md#user_list_tenant_invites) | **GET** /api/v1/users/invites | List tenant invites |


## alert_email_group_create

> <TenantAlertEmailGroup> alert_email_group_create(tenant, create_tenant_alert_email_group_request)

Create tenant alert email group

Creates a new tenant alert email group

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'
# setup authorization
HatchetSdkRest.configure do |config|
  # Configure API key authorization: cookieAuth
  config.api_key['hatchet'] = 'YOUR API KEY'
  # Uncomment the following line to set a prefix for the API key, e.g. 'Bearer' (defaults to nil)
  # config.api_key_prefix['hatchet'] = 'Bearer'

  # Configure Bearer authorization: bearerAuth
  config.access_token = 'YOUR_BEARER_TOKEN'
end

api_instance = HatchetSdkRest::TenantApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
create_tenant_alert_email_group_request = HatchetSdkRest::CreateTenantAlertEmailGroupRequest.new({emails: ['emails_example']}) # CreateTenantAlertEmailGroupRequest | The tenant alert email group to create

begin
  # Create tenant alert email group
  result = api_instance.alert_email_group_create(tenant, create_tenant_alert_email_group_request)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TenantApi->alert_email_group_create: #{e}"
end
```

#### Using the alert_email_group_create_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<TenantAlertEmailGroup>, Integer, Hash)> alert_email_group_create_with_http_info(tenant, create_tenant_alert_email_group_request)

```ruby
begin
  # Create tenant alert email group
  data, status_code, headers = api_instance.alert_email_group_create_with_http_info(tenant, create_tenant_alert_email_group_request)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <TenantAlertEmailGroup>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TenantApi->alert_email_group_create_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **create_tenant_alert_email_group_request** | [**CreateTenantAlertEmailGroupRequest**](CreateTenantAlertEmailGroupRequest.md) | The tenant alert email group to create |  |

### Return type

[**TenantAlertEmailGroup**](TenantAlertEmailGroup.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json


## alert_email_group_delete

> alert_email_group_delete(alert_email_group)

Delete tenant alert email group

Deletes a tenant alert email group

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'
# setup authorization
HatchetSdkRest.configure do |config|
  # Configure API key authorization: cookieAuth
  config.api_key['hatchet'] = 'YOUR API KEY'
  # Uncomment the following line to set a prefix for the API key, e.g. 'Bearer' (defaults to nil)
  # config.api_key_prefix['hatchet'] = 'Bearer'

  # Configure Bearer authorization: bearerAuth
  config.access_token = 'YOUR_BEARER_TOKEN'
end

api_instance = HatchetSdkRest::TenantApi.new
alert_email_group = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant alert email group id

begin
  # Delete tenant alert email group
  api_instance.alert_email_group_delete(alert_email_group)
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TenantApi->alert_email_group_delete: #{e}"
end
```

#### Using the alert_email_group_delete_with_http_info variant

This returns an Array which contains the response data (`nil` in this case), status code and headers.

> <Array(nil, Integer, Hash)> alert_email_group_delete_with_http_info(alert_email_group)

```ruby
begin
  # Delete tenant alert email group
  data, status_code, headers = api_instance.alert_email_group_delete_with_http_info(alert_email_group)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => nil
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TenantApi->alert_email_group_delete_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **alert_email_group** | **String** | The tenant alert email group id |  |

### Return type

nil (empty response body)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## alert_email_group_list

> <TenantAlertEmailGroupList> alert_email_group_list(tenant)

List tenant alert email groups

Gets a list of tenant alert email groups

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'
# setup authorization
HatchetSdkRest.configure do |config|
  # Configure API key authorization: cookieAuth
  config.api_key['hatchet'] = 'YOUR API KEY'
  # Uncomment the following line to set a prefix for the API key, e.g. 'Bearer' (defaults to nil)
  # config.api_key_prefix['hatchet'] = 'Bearer'

  # Configure Bearer authorization: bearerAuth
  config.access_token = 'YOUR_BEARER_TOKEN'
end

api_instance = HatchetSdkRest::TenantApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id

begin
  # List tenant alert email groups
  result = api_instance.alert_email_group_list(tenant)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TenantApi->alert_email_group_list: #{e}"
end
```

#### Using the alert_email_group_list_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<TenantAlertEmailGroupList>, Integer, Hash)> alert_email_group_list_with_http_info(tenant)

```ruby
begin
  # List tenant alert email groups
  data, status_code, headers = api_instance.alert_email_group_list_with_http_info(tenant)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <TenantAlertEmailGroupList>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TenantApi->alert_email_group_list_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |

### Return type

[**TenantAlertEmailGroupList**](TenantAlertEmailGroupList.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## alert_email_group_update

> <TenantAlertEmailGroup> alert_email_group_update(alert_email_group, update_tenant_alert_email_group_request)

Update tenant alert email group

Updates a tenant alert email group

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'
# setup authorization
HatchetSdkRest.configure do |config|
  # Configure API key authorization: cookieAuth
  config.api_key['hatchet'] = 'YOUR API KEY'
  # Uncomment the following line to set a prefix for the API key, e.g. 'Bearer' (defaults to nil)
  # config.api_key_prefix['hatchet'] = 'Bearer'

  # Configure Bearer authorization: bearerAuth
  config.access_token = 'YOUR_BEARER_TOKEN'
end

api_instance = HatchetSdkRest::TenantApi.new
alert_email_group = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant alert email group id
update_tenant_alert_email_group_request = HatchetSdkRest::UpdateTenantAlertEmailGroupRequest.new({emails: ['emails_example']}) # UpdateTenantAlertEmailGroupRequest | The tenant alert email group to update

begin
  # Update tenant alert email group
  result = api_instance.alert_email_group_update(alert_email_group, update_tenant_alert_email_group_request)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TenantApi->alert_email_group_update: #{e}"
end
```

#### Using the alert_email_group_update_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<TenantAlertEmailGroup>, Integer, Hash)> alert_email_group_update_with_http_info(alert_email_group, update_tenant_alert_email_group_request)

```ruby
begin
  # Update tenant alert email group
  data, status_code, headers = api_instance.alert_email_group_update_with_http_info(alert_email_group, update_tenant_alert_email_group_request)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <TenantAlertEmailGroup>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TenantApi->alert_email_group_update_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **alert_email_group** | **String** | The tenant alert email group id |  |
| **update_tenant_alert_email_group_request** | [**UpdateTenantAlertEmailGroupRequest**](UpdateTenantAlertEmailGroupRequest.md) | The tenant alert email group to update |  |

### Return type

[**TenantAlertEmailGroup**](TenantAlertEmailGroup.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json


## tenant_alerting_settings_get

> <TenantAlertingSettings> tenant_alerting_settings_get(tenant)

Get tenant alerting settings

Gets the alerting settings for a tenant

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'
# setup authorization
HatchetSdkRest.configure do |config|
  # Configure API key authorization: cookieAuth
  config.api_key['hatchet'] = 'YOUR API KEY'
  # Uncomment the following line to set a prefix for the API key, e.g. 'Bearer' (defaults to nil)
  # config.api_key_prefix['hatchet'] = 'Bearer'

  # Configure Bearer authorization: bearerAuth
  config.access_token = 'YOUR_BEARER_TOKEN'
end

api_instance = HatchetSdkRest::TenantApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id

begin
  # Get tenant alerting settings
  result = api_instance.tenant_alerting_settings_get(tenant)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TenantApi->tenant_alerting_settings_get: #{e}"
end
```

#### Using the tenant_alerting_settings_get_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<TenantAlertingSettings>, Integer, Hash)> tenant_alerting_settings_get_with_http_info(tenant)

```ruby
begin
  # Get tenant alerting settings
  data, status_code, headers = api_instance.tenant_alerting_settings_get_with_http_info(tenant)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <TenantAlertingSettings>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TenantApi->tenant_alerting_settings_get_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |

### Return type

[**TenantAlertingSettings**](TenantAlertingSettings.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## tenant_create

> <Tenant> tenant_create(create_tenant_request)

Create tenant

Creates a new tenant

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'
# setup authorization
HatchetSdkRest.configure do |config|
  # Configure API key authorization: cookieAuth
  config.api_key['hatchet'] = 'YOUR API KEY'
  # Uncomment the following line to set a prefix for the API key, e.g. 'Bearer' (defaults to nil)
  # config.api_key_prefix['hatchet'] = 'Bearer'

  # Configure Bearer authorization: bearerAuth
  config.access_token = 'YOUR_BEARER_TOKEN'
end

api_instance = HatchetSdkRest::TenantApi.new
create_tenant_request = HatchetSdkRest::CreateTenantRequest.new({name: 'name_example', slug: 'slug_example'}) # CreateTenantRequest | The tenant to create

begin
  # Create tenant
  result = api_instance.tenant_create(create_tenant_request)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TenantApi->tenant_create: #{e}"
end
```

#### Using the tenant_create_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<Tenant>, Integer, Hash)> tenant_create_with_http_info(create_tenant_request)

```ruby
begin
  # Create tenant
  data, status_code, headers = api_instance.tenant_create_with_http_info(create_tenant_request)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <Tenant>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TenantApi->tenant_create_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **create_tenant_request** | [**CreateTenantRequest**](CreateTenantRequest.md) | The tenant to create |  |

### Return type

[**Tenant**](Tenant.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json


## tenant_get

> <Tenant> tenant_get(tenant)

Get tenant

Get the details of a tenant

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'
# setup authorization
HatchetSdkRest.configure do |config|
  # Configure API key authorization: cookieAuth
  config.api_key['hatchet'] = 'YOUR API KEY'
  # Uncomment the following line to set a prefix for the API key, e.g. 'Bearer' (defaults to nil)
  # config.api_key_prefix['hatchet'] = 'Bearer'

  # Configure Bearer authorization: bearerAuth
  config.access_token = 'YOUR_BEARER_TOKEN'
end

api_instance = HatchetSdkRest::TenantApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id to get details for

begin
  # Get tenant
  result = api_instance.tenant_get(tenant)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TenantApi->tenant_get: #{e}"
end
```

#### Using the tenant_get_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<Tenant>, Integer, Hash)> tenant_get_with_http_info(tenant)

```ruby
begin
  # Get tenant
  data, status_code, headers = api_instance.tenant_get_with_http_info(tenant)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <Tenant>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TenantApi->tenant_get_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id to get details for |  |

### Return type

[**Tenant**](Tenant.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## tenant_get_prometheus_metrics

> String tenant_get_prometheus_metrics(tenant)

Get prometheus metrics

Get the prometheus metrics for the tenant

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'
# setup authorization
HatchetSdkRest.configure do |config|
  # Configure API key authorization: cookieAuth
  config.api_key['hatchet'] = 'YOUR API KEY'
  # Uncomment the following line to set a prefix for the API key, e.g. 'Bearer' (defaults to nil)
  # config.api_key_prefix['hatchet'] = 'Bearer'

  # Configure Bearer authorization: bearerAuth
  config.access_token = 'YOUR_BEARER_TOKEN'
end

api_instance = HatchetSdkRest::TenantApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id

begin
  # Get prometheus metrics
  result = api_instance.tenant_get_prometheus_metrics(tenant)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TenantApi->tenant_get_prometheus_metrics: #{e}"
end
```

#### Using the tenant_get_prometheus_metrics_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(String, Integer, Hash)> tenant_get_prometheus_metrics_with_http_info(tenant)

```ruby
begin
  # Get prometheus metrics
  data, status_code, headers = api_instance.tenant_get_prometheus_metrics_with_http_info(tenant)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => String
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TenantApi->tenant_get_prometheus_metrics_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |

### Return type

**String**

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: text/plain, application/json


## tenant_get_step_run_queue_metrics

> <TenantStepRunQueueMetrics> tenant_get_step_run_queue_metrics(tenant)

Get step run metrics

Get the queue metrics for the tenant

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'
# setup authorization
HatchetSdkRest.configure do |config|
  # Configure API key authorization: cookieAuth
  config.api_key['hatchet'] = 'YOUR API KEY'
  # Uncomment the following line to set a prefix for the API key, e.g. 'Bearer' (defaults to nil)
  # config.api_key_prefix['hatchet'] = 'Bearer'

  # Configure Bearer authorization: bearerAuth
  config.access_token = 'YOUR_BEARER_TOKEN'
end

api_instance = HatchetSdkRest::TenantApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id

begin
  # Get step run metrics
  result = api_instance.tenant_get_step_run_queue_metrics(tenant)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TenantApi->tenant_get_step_run_queue_metrics: #{e}"
end
```

#### Using the tenant_get_step_run_queue_metrics_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<TenantStepRunQueueMetrics>, Integer, Hash)> tenant_get_step_run_queue_metrics_with_http_info(tenant)

```ruby
begin
  # Get step run metrics
  data, status_code, headers = api_instance.tenant_get_step_run_queue_metrics_with_http_info(tenant)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <TenantStepRunQueueMetrics>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TenantApi->tenant_get_step_run_queue_metrics_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |

### Return type

[**TenantStepRunQueueMetrics**](TenantStepRunQueueMetrics.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## tenant_invite_accept

> tenant_invite_accept(opts)

Accept tenant invite

Accepts a tenant invite

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'
# setup authorization
HatchetSdkRest.configure do |config|
  # Configure API key authorization: cookieAuth
  config.api_key['hatchet'] = 'YOUR API KEY'
  # Uncomment the following line to set a prefix for the API key, e.g. 'Bearer' (defaults to nil)
  # config.api_key_prefix['hatchet'] = 'Bearer'

  # Configure Bearer authorization: bearerAuth
  config.access_token = 'YOUR_BEARER_TOKEN'
end

api_instance = HatchetSdkRest::TenantApi.new
opts = {
  accept_invite_request: HatchetSdkRest::AcceptInviteRequest.new({invite: 'bb214807-246e-43a5-a25d-41761d1cff9e'}) # AcceptInviteRequest | 
}

begin
  # Accept tenant invite
  api_instance.tenant_invite_accept(opts)
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TenantApi->tenant_invite_accept: #{e}"
end
```

#### Using the tenant_invite_accept_with_http_info variant

This returns an Array which contains the response data (`nil` in this case), status code and headers.

> <Array(nil, Integer, Hash)> tenant_invite_accept_with_http_info(opts)

```ruby
begin
  # Accept tenant invite
  data, status_code, headers = api_instance.tenant_invite_accept_with_http_info(opts)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => nil
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TenantApi->tenant_invite_accept_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **accept_invite_request** | [**AcceptInviteRequest**](AcceptInviteRequest.md) |  | [optional] |

### Return type

nil (empty response body)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json


## tenant_invite_create

> <TenantInvite> tenant_invite_create(tenant, create_tenant_invite_request)

Create tenant invite

Creates a new tenant invite

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'
# setup authorization
HatchetSdkRest.configure do |config|
  # Configure API key authorization: cookieAuth
  config.api_key['hatchet'] = 'YOUR API KEY'
  # Uncomment the following line to set a prefix for the API key, e.g. 'Bearer' (defaults to nil)
  # config.api_key_prefix['hatchet'] = 'Bearer'

  # Configure Bearer authorization: bearerAuth
  config.access_token = 'YOUR_BEARER_TOKEN'
end

api_instance = HatchetSdkRest::TenantApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
create_tenant_invite_request = HatchetSdkRest::CreateTenantInviteRequest.new({email: 'email_example', role: HatchetSdkRest::TenantMemberRole::OWNER}) # CreateTenantInviteRequest | The tenant invite to create

begin
  # Create tenant invite
  result = api_instance.tenant_invite_create(tenant, create_tenant_invite_request)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TenantApi->tenant_invite_create: #{e}"
end
```

#### Using the tenant_invite_create_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<TenantInvite>, Integer, Hash)> tenant_invite_create_with_http_info(tenant, create_tenant_invite_request)

```ruby
begin
  # Create tenant invite
  data, status_code, headers = api_instance.tenant_invite_create_with_http_info(tenant, create_tenant_invite_request)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <TenantInvite>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TenantApi->tenant_invite_create_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **create_tenant_invite_request** | [**CreateTenantInviteRequest**](CreateTenantInviteRequest.md) | The tenant invite to create |  |

### Return type

[**TenantInvite**](TenantInvite.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json


## tenant_invite_list

> <TenantInviteList> tenant_invite_list(tenant)

List tenant invites

Gets a list of tenant invites

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'
# setup authorization
HatchetSdkRest.configure do |config|
  # Configure API key authorization: cookieAuth
  config.api_key['hatchet'] = 'YOUR API KEY'
  # Uncomment the following line to set a prefix for the API key, e.g. 'Bearer' (defaults to nil)
  # config.api_key_prefix['hatchet'] = 'Bearer'

  # Configure Bearer authorization: bearerAuth
  config.access_token = 'YOUR_BEARER_TOKEN'
end

api_instance = HatchetSdkRest::TenantApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id

begin
  # List tenant invites
  result = api_instance.tenant_invite_list(tenant)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TenantApi->tenant_invite_list: #{e}"
end
```

#### Using the tenant_invite_list_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<TenantInviteList>, Integer, Hash)> tenant_invite_list_with_http_info(tenant)

```ruby
begin
  # List tenant invites
  data, status_code, headers = api_instance.tenant_invite_list_with_http_info(tenant)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <TenantInviteList>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TenantApi->tenant_invite_list_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |

### Return type

[**TenantInviteList**](TenantInviteList.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## tenant_invite_reject

> tenant_invite_reject(opts)

Reject tenant invite

Rejects a tenant invite

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'
# setup authorization
HatchetSdkRest.configure do |config|
  # Configure API key authorization: cookieAuth
  config.api_key['hatchet'] = 'YOUR API KEY'
  # Uncomment the following line to set a prefix for the API key, e.g. 'Bearer' (defaults to nil)
  # config.api_key_prefix['hatchet'] = 'Bearer'

  # Configure Bearer authorization: bearerAuth
  config.access_token = 'YOUR_BEARER_TOKEN'
end

api_instance = HatchetSdkRest::TenantApi.new
opts = {
  reject_invite_request: HatchetSdkRest::RejectInviteRequest.new({invite: 'bb214807-246e-43a5-a25d-41761d1cff9e'}) # RejectInviteRequest | 
}

begin
  # Reject tenant invite
  api_instance.tenant_invite_reject(opts)
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TenantApi->tenant_invite_reject: #{e}"
end
```

#### Using the tenant_invite_reject_with_http_info variant

This returns an Array which contains the response data (`nil` in this case), status code and headers.

> <Array(nil, Integer, Hash)> tenant_invite_reject_with_http_info(opts)

```ruby
begin
  # Reject tenant invite
  data, status_code, headers = api_instance.tenant_invite_reject_with_http_info(opts)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => nil
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TenantApi->tenant_invite_reject_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **reject_invite_request** | [**RejectInviteRequest**](RejectInviteRequest.md) |  | [optional] |

### Return type

nil (empty response body)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json


## tenant_member_delete

> <TenantMember> tenant_member_delete(tenant, member)

Delete a tenant member

Delete a member from a tenant

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'
# setup authorization
HatchetSdkRest.configure do |config|
  # Configure API key authorization: cookieAuth
  config.api_key['hatchet'] = 'YOUR API KEY'
  # Uncomment the following line to set a prefix for the API key, e.g. 'Bearer' (defaults to nil)
  # config.api_key_prefix['hatchet'] = 'Bearer'

  # Configure Bearer authorization: bearerAuth
  config.access_token = 'YOUR_BEARER_TOKEN'
end

api_instance = HatchetSdkRest::TenantApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
member = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant member id

begin
  # Delete a tenant member
  result = api_instance.tenant_member_delete(tenant, member)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TenantApi->tenant_member_delete: #{e}"
end
```

#### Using the tenant_member_delete_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<TenantMember>, Integer, Hash)> tenant_member_delete_with_http_info(tenant, member)

```ruby
begin
  # Delete a tenant member
  data, status_code, headers = api_instance.tenant_member_delete_with_http_info(tenant, member)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <TenantMember>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TenantApi->tenant_member_delete_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **member** | **String** | The tenant member id |  |

### Return type

[**TenantMember**](TenantMember.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## tenant_member_list

> <TenantMemberList> tenant_member_list(tenant)

List tenant members

Gets a list of tenant members

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'
# setup authorization
HatchetSdkRest.configure do |config|
  # Configure API key authorization: cookieAuth
  config.api_key['hatchet'] = 'YOUR API KEY'
  # Uncomment the following line to set a prefix for the API key, e.g. 'Bearer' (defaults to nil)
  # config.api_key_prefix['hatchet'] = 'Bearer'

  # Configure Bearer authorization: bearerAuth
  config.access_token = 'YOUR_BEARER_TOKEN'
end

api_instance = HatchetSdkRest::TenantApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id

begin
  # List tenant members
  result = api_instance.tenant_member_list(tenant)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TenantApi->tenant_member_list: #{e}"
end
```

#### Using the tenant_member_list_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<TenantMemberList>, Integer, Hash)> tenant_member_list_with_http_info(tenant)

```ruby
begin
  # List tenant members
  data, status_code, headers = api_instance.tenant_member_list_with_http_info(tenant)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <TenantMemberList>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TenantApi->tenant_member_list_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |

### Return type

[**TenantMemberList**](TenantMemberList.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## tenant_resource_policy_get

> <TenantResourcePolicy> tenant_resource_policy_get(tenant)

Create tenant alert email group

Gets the resource policy for a tenant

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'
# setup authorization
HatchetSdkRest.configure do |config|
  # Configure API key authorization: cookieAuth
  config.api_key['hatchet'] = 'YOUR API KEY'
  # Uncomment the following line to set a prefix for the API key, e.g. 'Bearer' (defaults to nil)
  # config.api_key_prefix['hatchet'] = 'Bearer'

  # Configure Bearer authorization: bearerAuth
  config.access_token = 'YOUR_BEARER_TOKEN'
end

api_instance = HatchetSdkRest::TenantApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id

begin
  # Create tenant alert email group
  result = api_instance.tenant_resource_policy_get(tenant)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TenantApi->tenant_resource_policy_get: #{e}"
end
```

#### Using the tenant_resource_policy_get_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<TenantResourcePolicy>, Integer, Hash)> tenant_resource_policy_get_with_http_info(tenant)

```ruby
begin
  # Create tenant alert email group
  data, status_code, headers = api_instance.tenant_resource_policy_get_with_http_info(tenant)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <TenantResourcePolicy>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TenantApi->tenant_resource_policy_get_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |

### Return type

[**TenantResourcePolicy**](TenantResourcePolicy.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## tenant_update

> <Tenant> tenant_update(tenant, update_tenant_request)

Update tenant

Update an existing tenant

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'
# setup authorization
HatchetSdkRest.configure do |config|
  # Configure API key authorization: cookieAuth
  config.api_key['hatchet'] = 'YOUR API KEY'
  # Uncomment the following line to set a prefix for the API key, e.g. 'Bearer' (defaults to nil)
  # config.api_key_prefix['hatchet'] = 'Bearer'

  # Configure Bearer authorization: bearerAuth
  config.access_token = 'YOUR_BEARER_TOKEN'
end

api_instance = HatchetSdkRest::TenantApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
update_tenant_request = HatchetSdkRest::UpdateTenantRequest.new # UpdateTenantRequest | The tenant properties to update

begin
  # Update tenant
  result = api_instance.tenant_update(tenant, update_tenant_request)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TenantApi->tenant_update: #{e}"
end
```

#### Using the tenant_update_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<Tenant>, Integer, Hash)> tenant_update_with_http_info(tenant, update_tenant_request)

```ruby
begin
  # Update tenant
  data, status_code, headers = api_instance.tenant_update_with_http_info(tenant, update_tenant_request)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <Tenant>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TenantApi->tenant_update_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **update_tenant_request** | [**UpdateTenantRequest**](UpdateTenantRequest.md) | The tenant properties to update |  |

### Return type

[**Tenant**](Tenant.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json


## user_list_tenant_invites

> <TenantInviteList> user_list_tenant_invites

List tenant invites

Lists all tenant invites for the current user

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'
# setup authorization
HatchetSdkRest.configure do |config|
  # Configure API key authorization: cookieAuth
  config.api_key['hatchet'] = 'YOUR API KEY'
  # Uncomment the following line to set a prefix for the API key, e.g. 'Bearer' (defaults to nil)
  # config.api_key_prefix['hatchet'] = 'Bearer'
end

api_instance = HatchetSdkRest::TenantApi.new

begin
  # List tenant invites
  result = api_instance.user_list_tenant_invites
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TenantApi->user_list_tenant_invites: #{e}"
end
```

#### Using the user_list_tenant_invites_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<TenantInviteList>, Integer, Hash)> user_list_tenant_invites_with_http_info

```ruby
begin
  # List tenant invites
  data, status_code, headers = api_instance.user_list_tenant_invites_with_http_info
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <TenantInviteList>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TenantApi->user_list_tenant_invites_with_http_info: #{e}"
end
```

### Parameters

This endpoint does not need any parameter.

### Return type

[**TenantInviteList**](TenantInviteList.md)

### Authorization

[cookieAuth](../README.md#cookieAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

