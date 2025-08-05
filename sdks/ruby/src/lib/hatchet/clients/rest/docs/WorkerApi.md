# HatchetSdkRest::WorkerApi

All URIs are relative to *http://localhost*

| Method | HTTP request | Description |
| ------ | ------------ | ----------- |
| [**worker_get**](WorkerApi.md#worker_get) | **GET** /api/v1/workers/{worker} | Get worker |
| [**worker_list**](WorkerApi.md#worker_list) | **GET** /api/v1/tenants/{tenant}/worker | Get workers |
| [**worker_update**](WorkerApi.md#worker_update) | **PATCH** /api/v1/workers/{worker} | Update worker |


## worker_get

> <Worker> worker_get(worker)

Get worker

Get a worker

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

api_instance = HatchetSdkRest::WorkerApi.new
worker = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The worker id

begin
  # Get worker
  result = api_instance.worker_get(worker)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkerApi->worker_get: #{e}"
end
```

#### Using the worker_get_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<Worker>, Integer, Hash)> worker_get_with_http_info(worker)

```ruby
begin
  # Get worker
  data, status_code, headers = api_instance.worker_get_with_http_info(worker)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <Worker>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkerApi->worker_get_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **worker** | **String** | The worker id |  |

### Return type

[**Worker**](Worker.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## worker_list

> <WorkerList> worker_list(tenant)

Get workers

Get all workers for a tenant

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

api_instance = HatchetSdkRest::WorkerApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id

begin
  # Get workers
  result = api_instance.worker_list(tenant)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkerApi->worker_list: #{e}"
end
```

#### Using the worker_list_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<WorkerList>, Integer, Hash)> worker_list_with_http_info(tenant)

```ruby
begin
  # Get workers
  data, status_code, headers = api_instance.worker_list_with_http_info(tenant)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <WorkerList>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkerApi->worker_list_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |

### Return type

[**WorkerList**](WorkerList.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## worker_update

> <Worker> worker_update(worker, update_worker_request)

Update worker

Update a worker

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

api_instance = HatchetSdkRest::WorkerApi.new
worker = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The worker id
update_worker_request = HatchetSdkRest::UpdateWorkerRequest.new # UpdateWorkerRequest | The worker update

begin
  # Update worker
  result = api_instance.worker_update(worker, update_worker_request)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkerApi->worker_update: #{e}"
end
```

#### Using the worker_update_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<Worker>, Integer, Hash)> worker_update_with_http_info(worker, update_worker_request)

```ruby
begin
  # Update worker
  data, status_code, headers = api_instance.worker_update_with_http_info(worker, update_worker_request)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <Worker>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkerApi->worker_update_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **worker** | **String** | The worker id |  |
| **update_worker_request** | [**UpdateWorkerRequest**](UpdateWorkerRequest.md) | The worker update |  |

### Return type

[**Worker**](Worker.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

