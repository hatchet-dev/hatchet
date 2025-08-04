# HatchetSdkRest::FilterApi

All URIs are relative to *http://localhost*

| Method | HTTP request | Description |
| ------ | ------------ | ----------- |
| [**v1_filter_create**](FilterApi.md#v1_filter_create) | **POST** /api/v1/stable/tenants/{tenant}/filters | Create a filter |
| [**v1_filter_delete**](FilterApi.md#v1_filter_delete) | **DELETE** /api/v1/stable/tenants/{tenant}/filters/{v1-filter} |  |
| [**v1_filter_get**](FilterApi.md#v1_filter_get) | **GET** /api/v1/stable/tenants/{tenant}/filters/{v1-filter} | Get a filter |
| [**v1_filter_list**](FilterApi.md#v1_filter_list) | **GET** /api/v1/stable/tenants/{tenant}/filters | List filters |
| [**v1_filter_update**](FilterApi.md#v1_filter_update) | **PATCH** /api/v1/stable/tenants/{tenant}/filters/{v1-filter} |  |


## v1_filter_create

> <V1Filter> v1_filter_create(tenant, v1_create_filter_request)

Create a filter

Create a new filter

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

api_instance = HatchetSdkRest::FilterApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
v1_create_filter_request = HatchetSdkRest::V1CreateFilterRequest.new({workflow_id: 'workflow_id_example', expression: 'expression_example', scope: 'scope_example'}) # V1CreateFilterRequest | The input to the filter creation

begin
  # Create a filter
  result = api_instance.v1_filter_create(tenant, v1_create_filter_request)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling FilterApi->v1_filter_create: #{e}"
end
```

#### Using the v1_filter_create_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<V1Filter>, Integer, Hash)> v1_filter_create_with_http_info(tenant, v1_create_filter_request)

```ruby
begin
  # Create a filter
  data, status_code, headers = api_instance.v1_filter_create_with_http_info(tenant, v1_create_filter_request)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <V1Filter>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling FilterApi->v1_filter_create_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **v1_create_filter_request** | [**V1CreateFilterRequest**](V1CreateFilterRequest.md) | The input to the filter creation |  |

### Return type

[**V1Filter**](V1Filter.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json


## v1_filter_delete

> <V1Filter> v1_filter_delete(tenant, v1_filter)



Delete a filter

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

api_instance = HatchetSdkRest::FilterApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
v1_filter = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The filter id to delete

begin
  
  result = api_instance.v1_filter_delete(tenant, v1_filter)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling FilterApi->v1_filter_delete: #{e}"
end
```

#### Using the v1_filter_delete_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<V1Filter>, Integer, Hash)> v1_filter_delete_with_http_info(tenant, v1_filter)

```ruby
begin
  
  data, status_code, headers = api_instance.v1_filter_delete_with_http_info(tenant, v1_filter)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <V1Filter>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling FilterApi->v1_filter_delete_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **v1_filter** | **String** | The filter id to delete |  |

### Return type

[**V1Filter**](V1Filter.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## v1_filter_get

> <V1Filter> v1_filter_get(tenant, v1_filter)

Get a filter

Get a filter by its id

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

api_instance = HatchetSdkRest::FilterApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
v1_filter = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The filter id

begin
  # Get a filter
  result = api_instance.v1_filter_get(tenant, v1_filter)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling FilterApi->v1_filter_get: #{e}"
end
```

#### Using the v1_filter_get_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<V1Filter>, Integer, Hash)> v1_filter_get_with_http_info(tenant, v1_filter)

```ruby
begin
  # Get a filter
  data, status_code, headers = api_instance.v1_filter_get_with_http_info(tenant, v1_filter)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <V1Filter>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling FilterApi->v1_filter_get_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **v1_filter** | **String** | The filter id |  |

### Return type

[**V1Filter**](V1Filter.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## v1_filter_list

> <V1FilterList> v1_filter_list(tenant, opts)

List filters

Lists all filters for a tenant.

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

api_instance = HatchetSdkRest::FilterApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
opts = {
  offset: 789, # Integer | The number to skip
  limit: 789, # Integer | The number to limit by
  workflow_ids: ['inner_example'], # Array<String> | The workflow ids to filter by
  scopes: ['inner_example'] # Array<String> | The scopes to subset candidate filters by
}

begin
  # List filters
  result = api_instance.v1_filter_list(tenant, opts)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling FilterApi->v1_filter_list: #{e}"
end
```

#### Using the v1_filter_list_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<V1FilterList>, Integer, Hash)> v1_filter_list_with_http_info(tenant, opts)

```ruby
begin
  # List filters
  data, status_code, headers = api_instance.v1_filter_list_with_http_info(tenant, opts)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <V1FilterList>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling FilterApi->v1_filter_list_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **offset** | **Integer** | The number to skip | [optional] |
| **limit** | **Integer** | The number to limit by | [optional] |
| **workflow_ids** | [**Array&lt;String&gt;**](String.md) | The workflow ids to filter by | [optional] |
| **scopes** | [**Array&lt;String&gt;**](String.md) | The scopes to subset candidate filters by | [optional] |

### Return type

[**V1FilterList**](V1FilterList.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## v1_filter_update

> <V1Filter> v1_filter_update(tenant, v1_filter, v1_update_filter_request)



Update a filter

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

api_instance = HatchetSdkRest::FilterApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
v1_filter = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The filter id to update
v1_update_filter_request = HatchetSdkRest::V1UpdateFilterRequest.new # V1UpdateFilterRequest | The input to the filter update

begin
  
  result = api_instance.v1_filter_update(tenant, v1_filter, v1_update_filter_request)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling FilterApi->v1_filter_update: #{e}"
end
```

#### Using the v1_filter_update_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<V1Filter>, Integer, Hash)> v1_filter_update_with_http_info(tenant, v1_filter, v1_update_filter_request)

```ruby
begin
  
  data, status_code, headers = api_instance.v1_filter_update_with_http_info(tenant, v1_filter, v1_update_filter_request)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <V1Filter>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling FilterApi->v1_filter_update_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **v1_filter** | **String** | The filter id to update |  |
| **v1_update_filter_request** | [**V1UpdateFilterRequest**](V1UpdateFilterRequest.md) | The input to the filter update |  |

### Return type

[**V1Filter**](V1Filter.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

