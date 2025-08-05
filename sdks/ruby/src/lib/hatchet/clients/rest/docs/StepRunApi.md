# HatchetSdkRest::StepRunApi

All URIs are relative to *http://localhost*

| Method | HTTP request | Description |
| ------ | ------------ | ----------- |
| [**step_run_get**](StepRunApi.md#step_run_get) | **GET** /api/v1/tenants/{tenant}/step-runs/{step-run} | Get step run |
| [**step_run_get_schema**](StepRunApi.md#step_run_get_schema) | **GET** /api/v1/tenants/{tenant}/step-runs/{step-run}/schema | Get step run schema |
| [**step_run_list_archives**](StepRunApi.md#step_run_list_archives) | **GET** /api/v1/step-runs/{step-run}/archives | List archives for step run |
| [**step_run_list_events**](StepRunApi.md#step_run_list_events) | **GET** /api/v1/step-runs/{step-run}/events | List events for step run |
| [**step_run_update_cancel**](StepRunApi.md#step_run_update_cancel) | **POST** /api/v1/tenants/{tenant}/step-runs/{step-run}/cancel | Attempts to cancel a step run |
| [**step_run_update_rerun**](StepRunApi.md#step_run_update_rerun) | **POST** /api/v1/tenants/{tenant}/step-runs/{step-run}/rerun | Rerun step run |
| [**workflow_run_list_step_run_events**](StepRunApi.md#workflow_run_list_step_run_events) | **GET** /api/v1/tenants/{tenant}/workflow-runs/{workflow-run}/step-run-events | List events for all step runs for a workflow run |


## step_run_get

> <StepRun> step_run_get(tenant, step_run)

Get step run

Get a step run by id

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

api_instance = HatchetSdkRest::StepRunApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
step_run = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The step run id

begin
  # Get step run
  result = api_instance.step_run_get(tenant, step_run)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling StepRunApi->step_run_get: #{e}"
end
```

#### Using the step_run_get_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<StepRun>, Integer, Hash)> step_run_get_with_http_info(tenant, step_run)

```ruby
begin
  # Get step run
  data, status_code, headers = api_instance.step_run_get_with_http_info(tenant, step_run)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <StepRun>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling StepRunApi->step_run_get_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **step_run** | **String** | The step run id |  |

### Return type

[**StepRun**](StepRun.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## step_run_get_schema

> Object step_run_get_schema(tenant, step_run)

Get step run schema

Get the schema for a step run

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

api_instance = HatchetSdkRest::StepRunApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
step_run = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The step run id

begin
  # Get step run schema
  result = api_instance.step_run_get_schema(tenant, step_run)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling StepRunApi->step_run_get_schema: #{e}"
end
```

#### Using the step_run_get_schema_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(Object, Integer, Hash)> step_run_get_schema_with_http_info(tenant, step_run)

```ruby
begin
  # Get step run schema
  data, status_code, headers = api_instance.step_run_get_schema_with_http_info(tenant, step_run)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => Object
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling StepRunApi->step_run_get_schema_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **step_run** | **String** | The step run id |  |

### Return type

**Object**

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## step_run_list_archives

> <StepRunArchiveList> step_run_list_archives(step_run, opts)

List archives for step run

List archives for a step run

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

api_instance = HatchetSdkRest::StepRunApi.new
step_run = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The step run id
opts = {
  offset: 789, # Integer | The number to skip
  limit: 789 # Integer | The number to limit by
}

begin
  # List archives for step run
  result = api_instance.step_run_list_archives(step_run, opts)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling StepRunApi->step_run_list_archives: #{e}"
end
```

#### Using the step_run_list_archives_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<StepRunArchiveList>, Integer, Hash)> step_run_list_archives_with_http_info(step_run, opts)

```ruby
begin
  # List archives for step run
  data, status_code, headers = api_instance.step_run_list_archives_with_http_info(step_run, opts)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <StepRunArchiveList>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling StepRunApi->step_run_list_archives_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **step_run** | **String** | The step run id |  |
| **offset** | **Integer** | The number to skip | [optional] |
| **limit** | **Integer** | The number to limit by | [optional] |

### Return type

[**StepRunArchiveList**](StepRunArchiveList.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## step_run_list_events

> <StepRunEventList> step_run_list_events(step_run, opts)

List events for step run

List events for a step run

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

api_instance = HatchetSdkRest::StepRunApi.new
step_run = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The step run id
opts = {
  offset: 789, # Integer | The number to skip
  limit: 789 # Integer | The number to limit by
}

begin
  # List events for step run
  result = api_instance.step_run_list_events(step_run, opts)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling StepRunApi->step_run_list_events: #{e}"
end
```

#### Using the step_run_list_events_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<StepRunEventList>, Integer, Hash)> step_run_list_events_with_http_info(step_run, opts)

```ruby
begin
  # List events for step run
  data, status_code, headers = api_instance.step_run_list_events_with_http_info(step_run, opts)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <StepRunEventList>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling StepRunApi->step_run_list_events_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **step_run** | **String** | The step run id |  |
| **offset** | **Integer** | The number to skip | [optional] |
| **limit** | **Integer** | The number to limit by | [optional] |

### Return type

[**StepRunEventList**](StepRunEventList.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## step_run_update_cancel

> <StepRun> step_run_update_cancel(tenant, step_run)

Attempts to cancel a step run

Attempts to cancel a step run

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

api_instance = HatchetSdkRest::StepRunApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
step_run = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The step run id

begin
  # Attempts to cancel a step run
  result = api_instance.step_run_update_cancel(tenant, step_run)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling StepRunApi->step_run_update_cancel: #{e}"
end
```

#### Using the step_run_update_cancel_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<StepRun>, Integer, Hash)> step_run_update_cancel_with_http_info(tenant, step_run)

```ruby
begin
  # Attempts to cancel a step run
  data, status_code, headers = api_instance.step_run_update_cancel_with_http_info(tenant, step_run)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <StepRun>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling StepRunApi->step_run_update_cancel_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **step_run** | **String** | The step run id |  |

### Return type

[**StepRun**](StepRun.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## step_run_update_rerun

> <StepRun> step_run_update_rerun(tenant, step_run, rerun_step_run_request)

Rerun step run

Reruns a step run

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

api_instance = HatchetSdkRest::StepRunApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
step_run = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The step run id
rerun_step_run_request = HatchetSdkRest::RerunStepRunRequest.new({input: 3.56}) # RerunStepRunRequest | The input to the rerun

begin
  # Rerun step run
  result = api_instance.step_run_update_rerun(tenant, step_run, rerun_step_run_request)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling StepRunApi->step_run_update_rerun: #{e}"
end
```

#### Using the step_run_update_rerun_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<StepRun>, Integer, Hash)> step_run_update_rerun_with_http_info(tenant, step_run, rerun_step_run_request)

```ruby
begin
  # Rerun step run
  data, status_code, headers = api_instance.step_run_update_rerun_with_http_info(tenant, step_run, rerun_step_run_request)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <StepRun>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling StepRunApi->step_run_update_rerun_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **step_run** | **String** | The step run id |  |
| **rerun_step_run_request** | [**RerunStepRunRequest**](RerunStepRunRequest.md) | The input to the rerun |  |

### Return type

[**StepRun**](StepRun.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json


## workflow_run_list_step_run_events

> <StepRunEventList> workflow_run_list_step_run_events(tenant, workflow_run, opts)

List events for all step runs for a workflow run

List events for all step runs for a workflow run

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

api_instance = HatchetSdkRest::StepRunApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
workflow_run = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The workflow run id
opts = {
  last_id: 56 # Integer | Last ID of the last event
}

begin
  # List events for all step runs for a workflow run
  result = api_instance.workflow_run_list_step_run_events(tenant, workflow_run, opts)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling StepRunApi->workflow_run_list_step_run_events: #{e}"
end
```

#### Using the workflow_run_list_step_run_events_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<StepRunEventList>, Integer, Hash)> workflow_run_list_step_run_events_with_http_info(tenant, workflow_run, opts)

```ruby
begin
  # List events for all step runs for a workflow run
  data, status_code, headers = api_instance.workflow_run_list_step_run_events_with_http_info(tenant, workflow_run, opts)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <StepRunEventList>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling StepRunApi->workflow_run_list_step_run_events_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **workflow_run** | **String** | The workflow run id |  |
| **last_id** | **Integer** | Last ID of the last event | [optional] |

### Return type

[**StepRunEventList**](StepRunEventList.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

