# HatchetSdkRest::LogApi

All URIs are relative to *http://localhost*

| Method | HTTP request | Description |
| ------ | ------------ | ----------- |
| [**log_line_list**](LogApi.md#log_line_list) | **GET** /api/v1/step-runs/{step-run}/logs | List log lines |
| [**v1_log_line_list**](LogApi.md#v1_log_line_list) | **GET** /api/v1/stable/tasks/{task}/logs | List log lines |


## log_line_list

> <LogLineList> log_line_list(step_run, opts)

List log lines

Lists log lines for a step run.

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

api_instance = HatchetSdkRest::LogApi.new
step_run = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The step run id
opts = {
  offset: 789, # Integer | The number to skip
  limit: 789, # Integer | The number to limit by
  levels: [HatchetSdkRest::LogLineLevel::DEBUG], # Array<LogLineLevel> | A list of levels to filter by
  search: 'search_example', # String | The search query to filter for
  order_by_field: HatchetSdkRest::LogLineOrderByField::CREATED_AT, # LogLineOrderByField | What to order by
  order_by_direction: HatchetSdkRest::LogLineOrderByDirection::ASC # LogLineOrderByDirection | The order direction
}

begin
  # List log lines
  result = api_instance.log_line_list(step_run, opts)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling LogApi->log_line_list: #{e}"
end
```

#### Using the log_line_list_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<LogLineList>, Integer, Hash)> log_line_list_with_http_info(step_run, opts)

```ruby
begin
  # List log lines
  data, status_code, headers = api_instance.log_line_list_with_http_info(step_run, opts)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <LogLineList>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling LogApi->log_line_list_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **step_run** | **String** | The step run id |  |
| **offset** | **Integer** | The number to skip | [optional] |
| **limit** | **Integer** | The number to limit by | [optional] |
| **levels** | [**Array&lt;LogLineLevel&gt;**](LogLineLevel.md) | A list of levels to filter by | [optional] |
| **search** | **String** | The search query to filter for | [optional] |
| **order_by_field** | [**LogLineOrderByField**](.md) | What to order by | [optional] |
| **order_by_direction** | [**LogLineOrderByDirection**](.md) | The order direction | [optional] |

### Return type

[**LogLineList**](LogLineList.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## v1_log_line_list

> <V1LogLineList> v1_log_line_list(task)

List log lines

Lists log lines for a task

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

api_instance = HatchetSdkRest::LogApi.new
task = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The task id

begin
  # List log lines
  result = api_instance.v1_log_line_list(task)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling LogApi->v1_log_line_list: #{e}"
end
```

#### Using the v1_log_line_list_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<V1LogLineList>, Integer, Hash)> v1_log_line_list_with_http_info(task)

```ruby
begin
  # List log lines
  data, status_code, headers = api_instance.v1_log_line_list_with_http_info(task)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <V1LogLineList>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling LogApi->v1_log_line_list_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **task** | **String** | The task id |  |

### Return type

[**V1LogLineList**](V1LogLineList.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

