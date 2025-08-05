# HatchetSdkRest::APITokenApi

All URIs are relative to *http://localhost*

| Method | HTTP request | Description |
| ------ | ------------ | ----------- |
| [**api_token_create**](APITokenApi.md#api_token_create) | **POST** /api/v1/tenants/{tenant}/api-tokens | Create API Token |
| [**api_token_list**](APITokenApi.md#api_token_list) | **GET** /api/v1/tenants/{tenant}/api-tokens | List API Tokens |
| [**api_token_update_revoke**](APITokenApi.md#api_token_update_revoke) | **POST** /api/v1/api-tokens/{api-token} | Revoke API Token |


## api_token_create

> <CreateAPITokenResponse> api_token_create(tenant, opts)

Create API Token

Create an API token for a tenant

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

api_instance = HatchetSdkRest::APITokenApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
opts = {
  create_api_token_request: HatchetSdkRest::CreateAPITokenRequest.new({name: 'name_example'}) # CreateAPITokenRequest | 
}

begin
  # Create API Token
  result = api_instance.api_token_create(tenant, opts)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling APITokenApi->api_token_create: #{e}"
end
```

#### Using the api_token_create_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<CreateAPITokenResponse>, Integer, Hash)> api_token_create_with_http_info(tenant, opts)

```ruby
begin
  # Create API Token
  data, status_code, headers = api_instance.api_token_create_with_http_info(tenant, opts)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <CreateAPITokenResponse>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling APITokenApi->api_token_create_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **create_api_token_request** | [**CreateAPITokenRequest**](CreateAPITokenRequest.md) |  | [optional] |

### Return type

[**CreateAPITokenResponse**](CreateAPITokenResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json


## api_token_list

> <ListAPITokensResponse> api_token_list(tenant)

List API Tokens

List API tokens for a tenant

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

api_instance = HatchetSdkRest::APITokenApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id

begin
  # List API Tokens
  result = api_instance.api_token_list(tenant)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling APITokenApi->api_token_list: #{e}"
end
```

#### Using the api_token_list_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<ListAPITokensResponse>, Integer, Hash)> api_token_list_with_http_info(tenant)

```ruby
begin
  # List API Tokens
  data, status_code, headers = api_instance.api_token_list_with_http_info(tenant)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <ListAPITokensResponse>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling APITokenApi->api_token_list_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |

### Return type

[**ListAPITokensResponse**](ListAPITokensResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## api_token_update_revoke

> api_token_update_revoke(api_token)

Revoke API Token

Revoke an API token for a tenant

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

api_instance = HatchetSdkRest::APITokenApi.new
api_token = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The API token

begin
  # Revoke API Token
  api_instance.api_token_update_revoke(api_token)
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling APITokenApi->api_token_update_revoke: #{e}"
end
```

#### Using the api_token_update_revoke_with_http_info variant

This returns an Array which contains the response data (`nil` in this case), status code and headers.

> <Array(nil, Integer, Hash)> api_token_update_revoke_with_http_info(api_token)

```ruby
begin
  # Revoke API Token
  data, status_code, headers = api_instance.api_token_update_revoke_with_http_info(api_token)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => nil
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling APITokenApi->api_token_update_revoke_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **api_token** | **String** | The API token |  |

### Return type

nil (empty response body)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

