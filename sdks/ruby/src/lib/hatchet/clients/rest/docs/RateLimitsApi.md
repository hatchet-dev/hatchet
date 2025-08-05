# HatchetSdkRest::RateLimitsApi

All URIs are relative to *http://localhost*

| Method | HTTP request | Description |
| ------ | ------------ | ----------- |
| [**rate_limit_list**](RateLimitsApi.md#rate_limit_list) | **GET** /api/v1/tenants/{tenant}/rate-limits | List rate limits |


## rate_limit_list

> <RateLimitList> rate_limit_list(tenant, opts)

List rate limits

Lists all rate limits for a tenant.

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

api_instance = HatchetSdkRest::RateLimitsApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
opts = {
  offset: 789, # Integer | The number to skip
  limit: 789, # Integer | The number to limit by
  search: 'search_example', # String | The search query to filter for
  order_by_field: HatchetSdkRest::RateLimitOrderByField::KEY, # RateLimitOrderByField | What to order by
  order_by_direction: HatchetSdkRest::RateLimitOrderByDirection::ASC # RateLimitOrderByDirection | The order direction
}

begin
  # List rate limits
  result = api_instance.rate_limit_list(tenant, opts)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling RateLimitsApi->rate_limit_list: #{e}"
end
```

#### Using the rate_limit_list_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<RateLimitList>, Integer, Hash)> rate_limit_list_with_http_info(tenant, opts)

```ruby
begin
  # List rate limits
  data, status_code, headers = api_instance.rate_limit_list_with_http_info(tenant, opts)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <RateLimitList>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling RateLimitsApi->rate_limit_list_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **offset** | **Integer** | The number to skip | [optional] |
| **limit** | **Integer** | The number to limit by | [optional] |
| **search** | **String** | The search query to filter for | [optional] |
| **order_by_field** | [**RateLimitOrderByField**](.md) | What to order by | [optional] |
| **order_by_direction** | [**RateLimitOrderByDirection**](.md) | The order direction | [optional] |

### Return type

[**RateLimitList**](RateLimitList.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

