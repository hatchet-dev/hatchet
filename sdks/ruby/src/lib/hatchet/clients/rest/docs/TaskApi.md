# HatchetSdkRest::TaskApi

All URIs are relative to *http://localhost*

| Method | HTTP request | Description |
| ------ | ------------ | ----------- |
| [**v1_dag_list_tasks**](TaskApi.md#v1_dag_list_tasks) | **GET** /api/v1/stable/dags/tasks | List tasks |
| [**v1_task_cancel**](TaskApi.md#v1_task_cancel) | **POST** /api/v1/stable/tenants/{tenant}/tasks/cancel | Cancel tasks |
| [**v1_task_event_list**](TaskApi.md#v1_task_event_list) | **GET** /api/v1/stable/tasks/{task}/task-events | List events for a task |
| [**v1_task_get**](TaskApi.md#v1_task_get) | **GET** /api/v1/stable/tasks/{task} | Get a task |
| [**v1_task_get_point_metrics**](TaskApi.md#v1_task_get_point_metrics) | **GET** /api/v1/stable/tenants/{tenant}/task-point-metrics | Get task point metrics |
| [**v1_task_list_status_metrics**](TaskApi.md#v1_task_list_status_metrics) | **GET** /api/v1/stable/tenants/{tenant}/task-metrics | Get task metrics |
| [**v1_task_replay**](TaskApi.md#v1_task_replay) | **POST** /api/v1/stable/tenants/{tenant}/tasks/replay | Replay tasks |


## v1_dag_list_tasks

> <Array<V1DagChildren>> v1_dag_list_tasks(dag_ids, tenant)

List tasks

Lists all tasks that belong a specific list of dags

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

api_instance = HatchetSdkRest::TaskApi.new
dag_ids = ['inner_example'] # Array<String> | The external id of the DAG
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id

begin
  # List tasks
  result = api_instance.v1_dag_list_tasks(dag_ids, tenant)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TaskApi->v1_dag_list_tasks: #{e}"
end
```

#### Using the v1_dag_list_tasks_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<Array<V1DagChildren>>, Integer, Hash)> v1_dag_list_tasks_with_http_info(dag_ids, tenant)

```ruby
begin
  # List tasks
  data, status_code, headers = api_instance.v1_dag_list_tasks_with_http_info(dag_ids, tenant)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <Array<V1DagChildren>>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TaskApi->v1_dag_list_tasks_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **dag_ids** | [**Array&lt;String&gt;**](String.md) | The external id of the DAG |  |
| **tenant** | **String** | The tenant id |  |

### Return type

[**Array&lt;V1DagChildren&gt;**](V1DagChildren.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## v1_task_cancel

> <V1CancelledTasks> v1_task_cancel(tenant, v1_cancel_task_request)

Cancel tasks

Cancel tasks

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

api_instance = HatchetSdkRest::TaskApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
v1_cancel_task_request = HatchetSdkRest::V1CancelTaskRequest.new # V1CancelTaskRequest | The tasks to cancel

begin
  # Cancel tasks
  result = api_instance.v1_task_cancel(tenant, v1_cancel_task_request)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TaskApi->v1_task_cancel: #{e}"
end
```

#### Using the v1_task_cancel_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<V1CancelledTasks>, Integer, Hash)> v1_task_cancel_with_http_info(tenant, v1_cancel_task_request)

```ruby
begin
  # Cancel tasks
  data, status_code, headers = api_instance.v1_task_cancel_with_http_info(tenant, v1_cancel_task_request)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <V1CancelledTasks>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TaskApi->v1_task_cancel_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **v1_cancel_task_request** | [**V1CancelTaskRequest**](V1CancelTaskRequest.md) | The tasks to cancel |  |

### Return type

[**V1CancelledTasks**](V1CancelledTasks.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json


## v1_task_event_list

> <V1TaskEventList> v1_task_event_list(task, opts)

List events for a task

List events for a task

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

api_instance = HatchetSdkRest::TaskApi.new
task = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The task id
opts = {
  offset: 789, # Integer | The number to skip
  limit: 789 # Integer | The number to limit by
}

begin
  # List events for a task
  result = api_instance.v1_task_event_list(task, opts)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TaskApi->v1_task_event_list: #{e}"
end
```

#### Using the v1_task_event_list_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<V1TaskEventList>, Integer, Hash)> v1_task_event_list_with_http_info(task, opts)

```ruby
begin
  # List events for a task
  data, status_code, headers = api_instance.v1_task_event_list_with_http_info(task, opts)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <V1TaskEventList>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TaskApi->v1_task_event_list_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **task** | **String** | The task id |  |
| **offset** | **Integer** | The number to skip | [optional] |
| **limit** | **Integer** | The number to limit by | [optional] |

### Return type

[**V1TaskEventList**](V1TaskEventList.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## v1_task_get

> <V1TaskSummary> v1_task_get(task, opts)

Get a task

Get a task by id

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

api_instance = HatchetSdkRest::TaskApi.new
task = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The task id
opts = {
  attempt: 56 # Integer | The attempt number
}

begin
  # Get a task
  result = api_instance.v1_task_get(task, opts)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TaskApi->v1_task_get: #{e}"
end
```

#### Using the v1_task_get_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<V1TaskSummary>, Integer, Hash)> v1_task_get_with_http_info(task, opts)

```ruby
begin
  # Get a task
  data, status_code, headers = api_instance.v1_task_get_with_http_info(task, opts)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <V1TaskSummary>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TaskApi->v1_task_get_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **task** | **String** | The task id |  |
| **attempt** | **Integer** | The attempt number | [optional] |

### Return type

[**V1TaskSummary**](V1TaskSummary.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## v1_task_get_point_metrics

> <V1TaskPointMetrics> v1_task_get_point_metrics(tenant, opts)

Get task point metrics

Get a minute by minute breakdown of task metrics for a tenant

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

api_instance = HatchetSdkRest::TaskApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
opts = {
  created_after: Time.parse('2021-01-01T00:00:00Z'), # Time | The time after the task was created
  finished_before: Time.parse('2021-01-01T00:00:00Z') # Time | The time before the task was completed
}

begin
  # Get task point metrics
  result = api_instance.v1_task_get_point_metrics(tenant, opts)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TaskApi->v1_task_get_point_metrics: #{e}"
end
```

#### Using the v1_task_get_point_metrics_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<V1TaskPointMetrics>, Integer, Hash)> v1_task_get_point_metrics_with_http_info(tenant, opts)

```ruby
begin
  # Get task point metrics
  data, status_code, headers = api_instance.v1_task_get_point_metrics_with_http_info(tenant, opts)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <V1TaskPointMetrics>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TaskApi->v1_task_get_point_metrics_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **created_after** | **Time** | The time after the task was created | [optional] |
| **finished_before** | **Time** | The time before the task was completed | [optional] |

### Return type

[**V1TaskPointMetrics**](V1TaskPointMetrics.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## v1_task_list_status_metrics

> <Array<V1TaskRunMetric>> v1_task_list_status_metrics(tenant, since, opts)

Get task metrics

Get a summary of task run metrics for a tenant

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

api_instance = HatchetSdkRest::TaskApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
since = Time.parse('2013-10-20T19:20:30+01:00') # Time | The start time to get metrics for
opts = {
  _until: Time.parse('2013-10-20T19:20:30+01:00'), # Time | The end time to get metrics for
  workflow_ids: ['inner_example'], # Array<String> | The workflow id to find runs for
  parent_task_external_id: '38400000-8cf0-11bd-b23e-10b96e4ef00d', # String | The parent task's external id
  triggering_event_external_id: '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The id of the event that triggered the task
}

begin
  # Get task metrics
  result = api_instance.v1_task_list_status_metrics(tenant, since, opts)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TaskApi->v1_task_list_status_metrics: #{e}"
end
```

#### Using the v1_task_list_status_metrics_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<Array<V1TaskRunMetric>>, Integer, Hash)> v1_task_list_status_metrics_with_http_info(tenant, since, opts)

```ruby
begin
  # Get task metrics
  data, status_code, headers = api_instance.v1_task_list_status_metrics_with_http_info(tenant, since, opts)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <Array<V1TaskRunMetric>>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TaskApi->v1_task_list_status_metrics_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **since** | **Time** | The start time to get metrics for |  |
| **_until** | **Time** | The end time to get metrics for | [optional] |
| **workflow_ids** | [**Array&lt;String&gt;**](String.md) | The workflow id to find runs for | [optional] |
| **parent_task_external_id** | **String** | The parent task&#39;s external id | [optional] |
| **triggering_event_external_id** | **String** | The id of the event that triggered the task | [optional] |

### Return type

[**Array&lt;V1TaskRunMetric&gt;**](V1TaskRunMetric.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## v1_task_replay

> <V1ReplayedTasks> v1_task_replay(tenant, v1_replay_task_request)

Replay tasks

Replay tasks

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

api_instance = HatchetSdkRest::TaskApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
v1_replay_task_request = HatchetSdkRest::V1ReplayTaskRequest.new # V1ReplayTaskRequest | The tasks to replay

begin
  # Replay tasks
  result = api_instance.v1_task_replay(tenant, v1_replay_task_request)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TaskApi->v1_task_replay: #{e}"
end
```

#### Using the v1_task_replay_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<V1ReplayedTasks>, Integer, Hash)> v1_task_replay_with_http_info(tenant, v1_replay_task_request)

```ruby
begin
  # Replay tasks
  data, status_code, headers = api_instance.v1_task_replay_with_http_info(tenant, v1_replay_task_request)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <V1ReplayedTasks>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling TaskApi->v1_task_replay_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **v1_replay_task_request** | [**V1ReplayTaskRequest**](V1ReplayTaskRequest.md) | The tasks to replay |  |

### Return type

[**V1ReplayedTasks**](V1ReplayedTasks.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

