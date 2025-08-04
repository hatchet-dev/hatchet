# HatchetSdkRest::DefaultApi

All URIs are relative to *http://localhost*

| Method | HTTP request | Description |
| ------ | ------------ | ----------- |
| [**info_get_version**](DefaultApi.md#info_get_version) | **GET** /api/v1/version | We return the version for the currently running server |
| [**monitoring_post_run_probe**](DefaultApi.md#monitoring_post_run_probe) | **POST** /api/v1/monitoring/{tenant}/probe | Detailed Health Probe For the Instance |
| [**tenant_invite_delete**](DefaultApi.md#tenant_invite_delete) | **DELETE** /api/v1/tenants/{tenant}/invites/{tenant-invite} | Delete invite |
| [**tenant_invite_update**](DefaultApi.md#tenant_invite_update) | **PATCH** /api/v1/tenants/{tenant}/invites/{tenant-invite} | Update invite |
| [**webhook_create**](DefaultApi.md#webhook_create) | **POST** /api/v1/tenants/{tenant}/webhook-workers | Create a webhook |
| [**webhook_delete**](DefaultApi.md#webhook_delete) | **DELETE** /api/v1/webhook-workers/{webhook} | Delete a webhook |
| [**webhook_list**](DefaultApi.md#webhook_list) | **GET** /api/v1/tenants/{tenant}/webhook-workers | List webhooks |
| [**webhook_requests_list**](DefaultApi.md#webhook_requests_list) | **GET** /api/v1/webhook-workers/{webhook}/requests | List webhook requests |


## info_get_version

> <InfoGetVersion200Response> info_get_version

We return the version for the currently running server

Get the version of the server

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'

api_instance = HatchetSdkRest::DefaultApi.new

begin
  # We return the version for the currently running server
  result = api_instance.info_get_version
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling DefaultApi->info_get_version: #{e}"
end
```

#### Using the info_get_version_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<InfoGetVersion200Response>, Integer, Hash)> info_get_version_with_http_info

```ruby
begin
  # We return the version for the currently running server
  data, status_code, headers = api_instance.info_get_version_with_http_info
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <InfoGetVersion200Response>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling DefaultApi->info_get_version_with_http_info: #{e}"
end
```

### Parameters

This endpoint does not need any parameter.

### Return type

[**InfoGetVersion200Response**](InfoGetVersion200Response.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## monitoring_post_run_probe

> monitoring_post_run_probe(tenant)

Detailed Health Probe For the Instance

Triggers a workflow to check the status of the instance

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

api_instance = HatchetSdkRest::DefaultApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id

begin
  # Detailed Health Probe For the Instance
  api_instance.monitoring_post_run_probe(tenant)
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling DefaultApi->monitoring_post_run_probe: #{e}"
end
```

#### Using the monitoring_post_run_probe_with_http_info variant

This returns an Array which contains the response data (`nil` in this case), status code and headers.

> <Array(nil, Integer, Hash)> monitoring_post_run_probe_with_http_info(tenant)

```ruby
begin
  # Detailed Health Probe For the Instance
  data, status_code, headers = api_instance.monitoring_post_run_probe_with_http_info(tenant)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => nil
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling DefaultApi->monitoring_post_run_probe_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |

### Return type

nil (empty response body)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## tenant_invite_delete

> <TenantInvite> tenant_invite_delete(tenant, tenant_invite)

Delete invite

Deletes a tenant invite

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

api_instance = HatchetSdkRest::DefaultApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
tenant_invite = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant invite id

begin
  # Delete invite
  result = api_instance.tenant_invite_delete(tenant, tenant_invite)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling DefaultApi->tenant_invite_delete: #{e}"
end
```

#### Using the tenant_invite_delete_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<TenantInvite>, Integer, Hash)> tenant_invite_delete_with_http_info(tenant, tenant_invite)

```ruby
begin
  # Delete invite
  data, status_code, headers = api_instance.tenant_invite_delete_with_http_info(tenant, tenant_invite)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <TenantInvite>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling DefaultApi->tenant_invite_delete_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **tenant_invite** | **String** | The tenant invite id |  |

### Return type

[**TenantInvite**](TenantInvite.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## tenant_invite_update

> <TenantInvite> tenant_invite_update(tenant, tenant_invite, update_tenant_invite_request)

Update invite

Updates a tenant invite

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

api_instance = HatchetSdkRest::DefaultApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
tenant_invite = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant invite id
update_tenant_invite_request = HatchetSdkRest::UpdateTenantInviteRequest.new({role: HatchetSdkRest::TenantMemberRole::OWNER}) # UpdateTenantInviteRequest | The tenant invite to update

begin
  # Update invite
  result = api_instance.tenant_invite_update(tenant, tenant_invite, update_tenant_invite_request)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling DefaultApi->tenant_invite_update: #{e}"
end
```

#### Using the tenant_invite_update_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<TenantInvite>, Integer, Hash)> tenant_invite_update_with_http_info(tenant, tenant_invite, update_tenant_invite_request)

```ruby
begin
  # Update invite
  data, status_code, headers = api_instance.tenant_invite_update_with_http_info(tenant, tenant_invite, update_tenant_invite_request)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <TenantInvite>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling DefaultApi->tenant_invite_update_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **tenant_invite** | **String** | The tenant invite id |  |
| **update_tenant_invite_request** | [**UpdateTenantInviteRequest**](UpdateTenantInviteRequest.md) | The tenant invite to update |  |

### Return type

[**TenantInvite**](TenantInvite.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json


## webhook_create

> <WebhookWorkerCreated> webhook_create(tenant, opts)

Create a webhook

Creates a webhook

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

api_instance = HatchetSdkRest::DefaultApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
opts = {
  webhook_worker_create_request: HatchetSdkRest::WebhookWorkerCreateRequest.new({name: 'name_example', url: 'url_example'}) # WebhookWorkerCreateRequest | 
}

begin
  # Create a webhook
  result = api_instance.webhook_create(tenant, opts)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling DefaultApi->webhook_create: #{e}"
end
```

#### Using the webhook_create_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<WebhookWorkerCreated>, Integer, Hash)> webhook_create_with_http_info(tenant, opts)

```ruby
begin
  # Create a webhook
  data, status_code, headers = api_instance.webhook_create_with_http_info(tenant, opts)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <WebhookWorkerCreated>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling DefaultApi->webhook_create_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **webhook_worker_create_request** | [**WebhookWorkerCreateRequest**](WebhookWorkerCreateRequest.md) |  | [optional] |

### Return type

[**WebhookWorkerCreated**](WebhookWorkerCreated.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json


## webhook_delete

> webhook_delete(webhook)

Delete a webhook

Deletes a webhook

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

api_instance = HatchetSdkRest::DefaultApi.new
webhook = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The webhook id

begin
  # Delete a webhook
  api_instance.webhook_delete(webhook)
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling DefaultApi->webhook_delete: #{e}"
end
```

#### Using the webhook_delete_with_http_info variant

This returns an Array which contains the response data (`nil` in this case), status code and headers.

> <Array(nil, Integer, Hash)> webhook_delete_with_http_info(webhook)

```ruby
begin
  # Delete a webhook
  data, status_code, headers = api_instance.webhook_delete_with_http_info(webhook)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => nil
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling DefaultApi->webhook_delete_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **webhook** | **String** | The webhook id |  |

### Return type

nil (empty response body)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## webhook_list

> <WebhookWorkerListResponse> webhook_list(tenant)

List webhooks

Lists all webhooks

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

api_instance = HatchetSdkRest::DefaultApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id

begin
  # List webhooks
  result = api_instance.webhook_list(tenant)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling DefaultApi->webhook_list: #{e}"
end
```

#### Using the webhook_list_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<WebhookWorkerListResponse>, Integer, Hash)> webhook_list_with_http_info(tenant)

```ruby
begin
  # List webhooks
  data, status_code, headers = api_instance.webhook_list_with_http_info(tenant)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <WebhookWorkerListResponse>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling DefaultApi->webhook_list_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |

### Return type

[**WebhookWorkerListResponse**](WebhookWorkerListResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## webhook_requests_list

> <WebhookWorkerRequestListResponse> webhook_requests_list(webhook)

List webhook requests

Lists all requests for a webhook

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

api_instance = HatchetSdkRest::DefaultApi.new
webhook = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The webhook id

begin
  # List webhook requests
  result = api_instance.webhook_requests_list(webhook)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling DefaultApi->webhook_requests_list: #{e}"
end
```

#### Using the webhook_requests_list_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<WebhookWorkerRequestListResponse>, Integer, Hash)> webhook_requests_list_with_http_info(webhook)

```ruby
begin
  # List webhook requests
  data, status_code, headers = api_instance.webhook_requests_list_with_http_info(webhook)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <WebhookWorkerRequestListResponse>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling DefaultApi->webhook_requests_list_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **webhook** | **String** | The webhook id |  |

### Return type

[**WebhookWorkerRequestListResponse**](WebhookWorkerRequestListResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

