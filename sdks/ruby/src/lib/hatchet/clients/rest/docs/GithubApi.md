# HatchetSdkRest::GithubApi

All URIs are relative to *http://localhost*

| Method | HTTP request | Description |
| ------ | ------------ | ----------- |
| [**sns_update**](GithubApi.md#sns_update) | **POST** /api/v1/sns/{tenant}/{event} | Github app tenant webhook |


## sns_update

> sns_update(tenant, event)

Github app tenant webhook

SNS event

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'

api_instance = HatchetSdkRest::GithubApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
event = 'event_example' # String | The event key

begin
  # Github app tenant webhook
  api_instance.sns_update(tenant, event)
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling GithubApi->sns_update: #{e}"
end
```

#### Using the sns_update_with_http_info variant

This returns an Array which contains the response data (`nil` in this case), status code and headers.

> <Array(nil, Integer, Hash)> sns_update_with_http_info(tenant, event)

```ruby
begin
  # Github app tenant webhook
  data, status_code, headers = api_instance.sns_update_with_http_info(tenant, event)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => nil
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling GithubApi->sns_update_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **event** | **String** | The event key |  |

### Return type

nil (empty response body)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

