# HatchetSdkRest::WorkflowRunsApi

All URIs are relative to *http://localhost*

| Method | HTTP request | Description |
| ------ | ------------ | ----------- |
| [**v1_workflow_run_create**](WorkflowRunsApi.md#v1_workflow_run_create) | **POST** /api/v1/stable/tenants/{tenant}/workflow-runs/trigger | Create workflow run |
| [**v1_workflow_run_display_names_list**](WorkflowRunsApi.md#v1_workflow_run_display_names_list) | **GET** /api/v1/stable/tenants/{tenant}/workflow-runs/display-names | List workflow runs |
| [**v1_workflow_run_get**](WorkflowRunsApi.md#v1_workflow_run_get) | **GET** /api/v1/stable/workflow-runs/{v1-workflow-run} | List tasks |
| [**v1_workflow_run_get_status**](WorkflowRunsApi.md#v1_workflow_run_get_status) | **GET** /api/v1/stable/workflow-runs/{v1-workflow-run}/status | Get workflow run status |
| [**v1_workflow_run_get_timings**](WorkflowRunsApi.md#v1_workflow_run_get_timings) | **GET** /api/v1/stable/workflow-runs/{v1-workflow-run}/task-timings | List timings for a workflow run |
| [**v1_workflow_run_list**](WorkflowRunsApi.md#v1_workflow_run_list) | **GET** /api/v1/stable/tenants/{tenant}/workflow-runs | List workflow runs |
| [**v1_workflow_run_task_events_list**](WorkflowRunsApi.md#v1_workflow_run_task_events_list) | **GET** /api/v1/stable/workflow-runs/{v1-workflow-run}/task-events | List tasks |


## v1_workflow_run_create

> <V1WorkflowRunDetails> v1_workflow_run_create(tenant, v1_trigger_workflow_run_request)

Create workflow run

Trigger a new workflow run

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

api_instance = HatchetSdkRest::WorkflowRunsApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
v1_trigger_workflow_run_request = HatchetSdkRest::V1TriggerWorkflowRunRequest.new({workflow_name: 'workflow_name_example', input: 3.56}) # V1TriggerWorkflowRunRequest | The workflow run to create

begin
  # Create workflow run
  result = api_instance.v1_workflow_run_create(tenant, v1_trigger_workflow_run_request)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowRunsApi->v1_workflow_run_create: #{e}"
end
```

#### Using the v1_workflow_run_create_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<V1WorkflowRunDetails>, Integer, Hash)> v1_workflow_run_create_with_http_info(tenant, v1_trigger_workflow_run_request)

```ruby
begin
  # Create workflow run
  data, status_code, headers = api_instance.v1_workflow_run_create_with_http_info(tenant, v1_trigger_workflow_run_request)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <V1WorkflowRunDetails>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowRunsApi->v1_workflow_run_create_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **v1_trigger_workflow_run_request** | [**V1TriggerWorkflowRunRequest**](V1TriggerWorkflowRunRequest.md) | The workflow run to create |  |

### Return type

[**V1WorkflowRunDetails**](V1WorkflowRunDetails.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json


## v1_workflow_run_display_names_list

> <V1WorkflowRunDisplayNameList> v1_workflow_run_display_names_list(tenant, external_ids)

List workflow runs

Lists displayable names of workflow runs for a tenant

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

api_instance = HatchetSdkRest::WorkflowRunsApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
external_ids = ['inner_example'] # Array<String> | The external ids of the workflow runs to get display names for

begin
  # List workflow runs
  result = api_instance.v1_workflow_run_display_names_list(tenant, external_ids)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowRunsApi->v1_workflow_run_display_names_list: #{e}"
end
```

#### Using the v1_workflow_run_display_names_list_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<V1WorkflowRunDisplayNameList>, Integer, Hash)> v1_workflow_run_display_names_list_with_http_info(tenant, external_ids)

```ruby
begin
  # List workflow runs
  data, status_code, headers = api_instance.v1_workflow_run_display_names_list_with_http_info(tenant, external_ids)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <V1WorkflowRunDisplayNameList>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowRunsApi->v1_workflow_run_display_names_list_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **external_ids** | [**Array&lt;String&gt;**](String.md) | The external ids of the workflow runs to get display names for |  |

### Return type

[**V1WorkflowRunDisplayNameList**](V1WorkflowRunDisplayNameList.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## v1_workflow_run_get

> <V1WorkflowRunDetails> v1_workflow_run_get(v1_workflow_run)

List tasks

Get a workflow run and its metadata to display on the \"detail\" page

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

api_instance = HatchetSdkRest::WorkflowRunsApi.new
v1_workflow_run = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The workflow run id to get

begin
  # List tasks
  result = api_instance.v1_workflow_run_get(v1_workflow_run)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowRunsApi->v1_workflow_run_get: #{e}"
end
```

#### Using the v1_workflow_run_get_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<V1WorkflowRunDetails>, Integer, Hash)> v1_workflow_run_get_with_http_info(v1_workflow_run)

```ruby
begin
  # List tasks
  data, status_code, headers = api_instance.v1_workflow_run_get_with_http_info(v1_workflow_run)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <V1WorkflowRunDetails>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowRunsApi->v1_workflow_run_get_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **v1_workflow_run** | **String** | The workflow run id to get |  |

### Return type

[**V1WorkflowRunDetails**](V1WorkflowRunDetails.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## v1_workflow_run_get_status

> <V1TaskStatus> v1_workflow_run_get_status(v1_workflow_run)

Get workflow run status

Get the status of a workflow run.

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

api_instance = HatchetSdkRest::WorkflowRunsApi.new
v1_workflow_run = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The workflow run id to get the status for

begin
  # Get workflow run status
  result = api_instance.v1_workflow_run_get_status(v1_workflow_run)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowRunsApi->v1_workflow_run_get_status: #{e}"
end
```

#### Using the v1_workflow_run_get_status_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<V1TaskStatus>, Integer, Hash)> v1_workflow_run_get_status_with_http_info(v1_workflow_run)

```ruby
begin
  # Get workflow run status
  data, status_code, headers = api_instance.v1_workflow_run_get_status_with_http_info(v1_workflow_run)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <V1TaskStatus>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowRunsApi->v1_workflow_run_get_status_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **v1_workflow_run** | **String** | The workflow run id to get the status for |  |

### Return type

[**V1TaskStatus**](V1TaskStatus.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## v1_workflow_run_get_timings

> <V1TaskTimingList> v1_workflow_run_get_timings(v1_workflow_run, opts)

List timings for a workflow run

Get the timings for a workflow run

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

api_instance = HatchetSdkRest::WorkflowRunsApi.new
v1_workflow_run = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The workflow run id to get
opts = {
  depth: 789 # Integer | The depth to retrieve children
}

begin
  # List timings for a workflow run
  result = api_instance.v1_workflow_run_get_timings(v1_workflow_run, opts)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowRunsApi->v1_workflow_run_get_timings: #{e}"
end
```

#### Using the v1_workflow_run_get_timings_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<V1TaskTimingList>, Integer, Hash)> v1_workflow_run_get_timings_with_http_info(v1_workflow_run, opts)

```ruby
begin
  # List timings for a workflow run
  data, status_code, headers = api_instance.v1_workflow_run_get_timings_with_http_info(v1_workflow_run, opts)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <V1TaskTimingList>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowRunsApi->v1_workflow_run_get_timings_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **v1_workflow_run** | **String** | The workflow run id to get |  |
| **depth** | **Integer** | The depth to retrieve children | [optional] |

### Return type

[**V1TaskTimingList**](V1TaskTimingList.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## v1_workflow_run_list

> <V1TaskSummaryList> v1_workflow_run_list(tenant, since, only_tasks, opts)

List workflow runs

Lists workflow runs for a tenant.

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

api_instance = HatchetSdkRest::WorkflowRunsApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
since = Time.parse('2013-10-20T19:20:30+01:00') # Time | The earliest date to filter by
only_tasks = true # Boolean | Whether to include DAGs or only to include tasks
opts = {
  offset: 789, # Integer | The number to skip
  limit: 789, # Integer | The number to limit by
  statuses: [HatchetSdkRest::V1TaskStatus::QUEUED], # Array<V1TaskStatus> | A list of statuses to filter by
  _until: Time.parse('2013-10-20T19:20:30+01:00'), # Time | The latest date to filter by
  additional_metadata: ['inner_example'], # Array<String> | Additional metadata k-v pairs to filter by
  workflow_ids: ['inner_example'], # Array<String> | The workflow ids to find runs for
  worker_id: '38400000-8cf0-11bd-b23e-10b96e4ef00d', # String | The worker id to filter by
  parent_task_external_id: '38400000-8cf0-11bd-b23e-10b96e4ef00d', # String | The parent task external id to filter by
  triggering_event_external_id: '38400000-8cf0-11bd-b23e-10b96e4ef00d', # String | The external id of the event that triggered the workflow run
  include_payloads: true # Boolean | A flag for whether or not to include the input and output payloads in the response. Defaults to `true` if unset.
}

begin
  # List workflow runs
  result = api_instance.v1_workflow_run_list(tenant, since, only_tasks, opts)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowRunsApi->v1_workflow_run_list: #{e}"
end
```

#### Using the v1_workflow_run_list_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<V1TaskSummaryList>, Integer, Hash)> v1_workflow_run_list_with_http_info(tenant, since, only_tasks, opts)

```ruby
begin
  # List workflow runs
  data, status_code, headers = api_instance.v1_workflow_run_list_with_http_info(tenant, since, only_tasks, opts)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <V1TaskSummaryList>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowRunsApi->v1_workflow_run_list_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **since** | **Time** | The earliest date to filter by |  |
| **only_tasks** | **Boolean** | Whether to include DAGs or only to include tasks |  |
| **offset** | **Integer** | The number to skip | [optional] |
| **limit** | **Integer** | The number to limit by | [optional] |
| **statuses** | [**Array&lt;V1TaskStatus&gt;**](V1TaskStatus.md) | A list of statuses to filter by | [optional] |
| **_until** | **Time** | The latest date to filter by | [optional] |
| **additional_metadata** | [**Array&lt;String&gt;**](String.md) | Additional metadata k-v pairs to filter by | [optional] |
| **workflow_ids** | [**Array&lt;String&gt;**](String.md) | The workflow ids to find runs for | [optional] |
| **worker_id** | **String** | The worker id to filter by | [optional] |
| **parent_task_external_id** | **String** | The parent task external id to filter by | [optional] |
| **triggering_event_external_id** | **String** | The external id of the event that triggered the workflow run | [optional] |
| **include_payloads** | **Boolean** | A flag for whether or not to include the input and output payloads in the response. Defaults to &#x60;true&#x60; if unset. | [optional] |

### Return type

[**V1TaskSummaryList**](V1TaskSummaryList.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## v1_workflow_run_task_events_list

> <V1TaskEventList> v1_workflow_run_task_events_list(v1_workflow_run, opts)

List tasks

List all tasks for a workflow run

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

api_instance = HatchetSdkRest::WorkflowRunsApi.new
v1_workflow_run = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The workflow run id to find runs for
opts = {
  offset: 789, # Integer | The number to skip
  limit: 789 # Integer | The number to limit by
}

begin
  # List tasks
  result = api_instance.v1_workflow_run_task_events_list(v1_workflow_run, opts)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowRunsApi->v1_workflow_run_task_events_list: #{e}"
end
```

#### Using the v1_workflow_run_task_events_list_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<V1TaskEventList>, Integer, Hash)> v1_workflow_run_task_events_list_with_http_info(v1_workflow_run, opts)

```ruby
begin
  # List tasks
  data, status_code, headers = api_instance.v1_workflow_run_task_events_list_with_http_info(v1_workflow_run, opts)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <V1TaskEventList>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowRunsApi->v1_workflow_run_task_events_list_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **v1_workflow_run** | **String** | The workflow run id to find runs for |  |
| **offset** | **Integer** | The number to skip | [optional] |
| **limit** | **Integer** | The number to limit by | [optional] |

### Return type

[**V1TaskEventList**](V1TaskEventList.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

