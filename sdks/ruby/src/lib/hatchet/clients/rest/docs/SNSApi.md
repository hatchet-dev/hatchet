# HatchetSdkRest::SNSApi

All URIs are relative to *http://localhost*

| Method | HTTP request | Description |
| ------ | ------------ | ----------- |
| [**sns_create**](SNSApi.md#sns_create) | **POST** /api/v1/tenants/{tenant}/sns | Create SNS integration |
| [**sns_delete**](SNSApi.md#sns_delete) | **DELETE** /api/v1/sns/{sns} | Delete SNS integration |
| [**sns_list**](SNSApi.md#sns_list) | **GET** /api/v1/tenants/{tenant}/sns | List SNS integrations |


## sns_create

> <SNSIntegration> sns_create(tenant, opts)

Create SNS integration

Create SNS integration

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

api_instance = HatchetSdkRest::SNSApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
opts = {
  create_sns_integration_request: HatchetSdkRest::CreateSNSIntegrationRequest.new({topic_arn: 'topic_arn_example'}) # CreateSNSIntegrationRequest | 
}

begin
  # Create SNS integration
  result = api_instance.sns_create(tenant, opts)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling SNSApi->sns_create: #{e}"
end
```

#### Using the sns_create_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<SNSIntegration>, Integer, Hash)> sns_create_with_http_info(tenant, opts)

```ruby
begin
  # Create SNS integration
  data, status_code, headers = api_instance.sns_create_with_http_info(tenant, opts)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <SNSIntegration>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling SNSApi->sns_create_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **create_sns_integration_request** | [**CreateSNSIntegrationRequest**](CreateSNSIntegrationRequest.md) |  | [optional] |

### Return type

[**SNSIntegration**](SNSIntegration.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json


## sns_delete

> sns_delete(sns)

Delete SNS integration

Delete SNS integration

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

api_instance = HatchetSdkRest::SNSApi.new
sns = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The SNS integration id

begin
  # Delete SNS integration
  api_instance.sns_delete(sns)
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling SNSApi->sns_delete: #{e}"
end
```

#### Using the sns_delete_with_http_info variant

This returns an Array which contains the response data (`nil` in this case), status code and headers.

> <Array(nil, Integer, Hash)> sns_delete_with_http_info(sns)

```ruby
begin
  # Delete SNS integration
  data, status_code, headers = api_instance.sns_delete_with_http_info(sns)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => nil
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling SNSApi->sns_delete_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **sns** | **String** | The SNS integration id |  |

### Return type

nil (empty response body)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## sns_list

> <ListSNSIntegrations> sns_list(tenant)

List SNS integrations

List SNS integrations

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

api_instance = HatchetSdkRest::SNSApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id

begin
  # List SNS integrations
  result = api_instance.sns_list(tenant)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling SNSApi->sns_list: #{e}"
end
```

#### Using the sns_list_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<ListSNSIntegrations>, Integer, Hash)> sns_list_with_http_info(tenant)

```ruby
begin
  # List SNS integrations
  data, status_code, headers = api_instance.sns_list_with_http_info(tenant)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <ListSNSIntegrations>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling SNSApi->sns_list_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |

### Return type

[**ListSNSIntegrations**](ListSNSIntegrations.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

