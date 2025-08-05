# HatchetSdkRest::MetadataApi

All URIs are relative to *http://localhost*

| Method | HTTP request | Description |
| ------ | ------------ | ----------- |
| [**cloud_metadata_get**](MetadataApi.md#cloud_metadata_get) | **GET** /api/v1/cloud/metadata | Get cloud metadata |
| [**metadata_get**](MetadataApi.md#metadata_get) | **GET** /api/v1/meta | Get metadata |
| [**metadata_list_integrations**](MetadataApi.md#metadata_list_integrations) | **GET** /api/v1/meta/integrations | List integrations |


## cloud_metadata_get

> <APIErrors> cloud_metadata_get

Get cloud metadata

Gets metadata for the Hatchet cloud instance

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'

api_instance = HatchetSdkRest::MetadataApi.new

begin
  # Get cloud metadata
  result = api_instance.cloud_metadata_get
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling MetadataApi->cloud_metadata_get: #{e}"
end
```

#### Using the cloud_metadata_get_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<APIErrors>, Integer, Hash)> cloud_metadata_get_with_http_info

```ruby
begin
  # Get cloud metadata
  data, status_code, headers = api_instance.cloud_metadata_get_with_http_info
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <APIErrors>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling MetadataApi->cloud_metadata_get_with_http_info: #{e}"
end
```

### Parameters

This endpoint does not need any parameter.

### Return type

[**APIErrors**](APIErrors.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## metadata_get

> <APIMeta> metadata_get

Get metadata

Gets metadata for the Hatchet instance

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'

api_instance = HatchetSdkRest::MetadataApi.new

begin
  # Get metadata
  result = api_instance.metadata_get
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling MetadataApi->metadata_get: #{e}"
end
```

#### Using the metadata_get_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<APIMeta>, Integer, Hash)> metadata_get_with_http_info

```ruby
begin
  # Get metadata
  data, status_code, headers = api_instance.metadata_get_with_http_info
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <APIMeta>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling MetadataApi->metadata_get_with_http_info: #{e}"
end
```

### Parameters

This endpoint does not need any parameter.

### Return type

[**APIMeta**](APIMeta.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## metadata_list_integrations

> <Array<APIMetaIntegration>> metadata_list_integrations

List integrations

List all integrations

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

api_instance = HatchetSdkRest::MetadataApi.new

begin
  # List integrations
  result = api_instance.metadata_list_integrations
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling MetadataApi->metadata_list_integrations: #{e}"
end
```

#### Using the metadata_list_integrations_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<Array<APIMetaIntegration>>, Integer, Hash)> metadata_list_integrations_with_http_info

```ruby
begin
  # List integrations
  data, status_code, headers = api_instance.metadata_list_integrations_with_http_info
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <Array<APIMetaIntegration>>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling MetadataApi->metadata_list_integrations_with_http_info: #{e}"
end
```

### Parameters

This endpoint does not need any parameter.

### Return type

[**Array&lt;APIMetaIntegration&gt;**](APIMetaIntegration.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

