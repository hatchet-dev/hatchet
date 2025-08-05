# HatchetSdkRest::SlackApi

All URIs are relative to *http://localhost*

| Method | HTTP request | Description |
| ------ | ------------ | ----------- |
| [**slack_webhook_delete**](SlackApi.md#slack_webhook_delete) | **DELETE** /api/v1/slack/{slack} | Delete Slack webhook |
| [**slack_webhook_list**](SlackApi.md#slack_webhook_list) | **GET** /api/v1/tenants/{tenant}/slack | List Slack integrations |


## slack_webhook_delete

> slack_webhook_delete(slack)

Delete Slack webhook

Delete Slack webhook

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

api_instance = HatchetSdkRest::SlackApi.new
slack = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The Slack webhook id

begin
  # Delete Slack webhook
  api_instance.slack_webhook_delete(slack)
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling SlackApi->slack_webhook_delete: #{e}"
end
```

#### Using the slack_webhook_delete_with_http_info variant

This returns an Array which contains the response data (`nil` in this case), status code and headers.

> <Array(nil, Integer, Hash)> slack_webhook_delete_with_http_info(slack)

```ruby
begin
  # Delete Slack webhook
  data, status_code, headers = api_instance.slack_webhook_delete_with_http_info(slack)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => nil
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling SlackApi->slack_webhook_delete_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **slack** | **String** | The Slack webhook id |  |

### Return type

nil (empty response body)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## slack_webhook_list

> <ListSlackWebhooks> slack_webhook_list(tenant)

List Slack integrations

List Slack webhooks

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

api_instance = HatchetSdkRest::SlackApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id

begin
  # List Slack integrations
  result = api_instance.slack_webhook_list(tenant)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling SlackApi->slack_webhook_list: #{e}"
end
```

#### Using the slack_webhook_list_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<ListSlackWebhooks>, Integer, Hash)> slack_webhook_list_with_http_info(tenant)

```ruby
begin
  # List Slack integrations
  data, status_code, headers = api_instance.slack_webhook_list_with_http_info(tenant)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <ListSlackWebhooks>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling SlackApi->slack_webhook_list_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |

### Return type

[**ListSlackWebhooks**](ListSlackWebhooks.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

