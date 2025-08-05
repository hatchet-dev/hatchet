# HatchetSdkRest::HealthcheckApi

All URIs are relative to *http://localhost*

| Method | HTTP request | Description |
| ------ | ------------ | ----------- |
| [**liveness_get**](HealthcheckApi.md#liveness_get) | **GET** /api/live | Get liveness |
| [**readiness_get**](HealthcheckApi.md#readiness_get) | **GET** /api/ready | Get readiness |


## liveness_get

> liveness_get

Get liveness

Gets the liveness status

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'

api_instance = HatchetSdkRest::HealthcheckApi.new

begin
  # Get liveness
  api_instance.liveness_get
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling HealthcheckApi->liveness_get: #{e}"
end
```

#### Using the liveness_get_with_http_info variant

This returns an Array which contains the response data (`nil` in this case), status code and headers.

> <Array(nil, Integer, Hash)> liveness_get_with_http_info

```ruby
begin
  # Get liveness
  data, status_code, headers = api_instance.liveness_get_with_http_info
  p status_code # => 2xx
  p headers # => { ... }
  p data # => nil
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling HealthcheckApi->liveness_get_with_http_info: #{e}"
end
```

### Parameters

This endpoint does not need any parameter.

### Return type

nil (empty response body)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: Not defined


## readiness_get

> readiness_get

Get readiness

Gets the readiness status

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'

api_instance = HatchetSdkRest::HealthcheckApi.new

begin
  # Get readiness
  api_instance.readiness_get
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling HealthcheckApi->readiness_get: #{e}"
end
```

#### Using the readiness_get_with_http_info variant

This returns an Array which contains the response data (`nil` in this case), status code and headers.

> <Array(nil, Integer, Hash)> readiness_get_with_http_info

```ruby
begin
  # Get readiness
  data, status_code, headers = api_instance.readiness_get_with_http_info
  p status_code # => 2xx
  p headers # => { ... }
  p data # => nil
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling HealthcheckApi->readiness_get_with_http_info: #{e}"
end
```

### Parameters

This endpoint does not need any parameter.

### Return type

nil (empty response body)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: Not defined

