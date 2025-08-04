# HatchetSdkRest::WorkflowApi

All URIs are relative to *http://localhost*

| Method | HTTP request | Description |
| ------ | ------------ | ----------- |
| [**cron_workflow_list**](WorkflowApi.md#cron_workflow_list) | **GET** /api/v1/tenants/{tenant}/workflows/crons | Get cron job workflows |
| [**tenant_get_queue_metrics**](WorkflowApi.md#tenant_get_queue_metrics) | **GET** /api/v1/tenants/{tenant}/queue-metrics | Get workflow metrics |
| [**workflow_cron_delete**](WorkflowApi.md#workflow_cron_delete) | **DELETE** /api/v1/tenants/{tenant}/workflows/crons/{cron-workflow} | Delete cron job workflow run |
| [**workflow_cron_get**](WorkflowApi.md#workflow_cron_get) | **GET** /api/v1/tenants/{tenant}/workflows/crons/{cron-workflow} | Get cron job workflow run |
| [**workflow_delete**](WorkflowApi.md#workflow_delete) | **DELETE** /api/v1/workflows/{workflow} | Delete workflow |
| [**workflow_get**](WorkflowApi.md#workflow_get) | **GET** /api/v1/workflows/{workflow} | Get workflow |
| [**workflow_get_metrics**](WorkflowApi.md#workflow_get_metrics) | **GET** /api/v1/workflows/{workflow}/metrics | Get workflow metrics |
| [**workflow_get_workers_count**](WorkflowApi.md#workflow_get_workers_count) | **GET** /api/v1/tenants/{tenant}/workflows/{workflow}/worker-count | Get workflow worker count |
| [**workflow_list**](WorkflowApi.md#workflow_list) | **GET** /api/v1/tenants/{tenant}/workflows | Get workflows |
| [**workflow_run_get**](WorkflowApi.md#workflow_run_get) | **GET** /api/v1/tenants/{tenant}/workflow-runs/{workflow-run} | Get workflow run |
| [**workflow_run_get_metrics**](WorkflowApi.md#workflow_run_get_metrics) | **GET** /api/v1/tenants/{tenant}/workflows/runs/metrics | Get workflow runs metrics |
| [**workflow_run_get_shape**](WorkflowApi.md#workflow_run_get_shape) | **GET** /api/v1/tenants/{tenant}/workflow-runs/{workflow-run}/shape | Get workflow run |
| [**workflow_run_list**](WorkflowApi.md#workflow_run_list) | **GET** /api/v1/tenants/{tenant}/workflows/runs | Get workflow runs |
| [**workflow_scheduled_delete**](WorkflowApi.md#workflow_scheduled_delete) | **DELETE** /api/v1/tenants/{tenant}/workflows/scheduled/{scheduled-workflow-run} | Delete scheduled workflow run |
| [**workflow_scheduled_get**](WorkflowApi.md#workflow_scheduled_get) | **GET** /api/v1/tenants/{tenant}/workflows/scheduled/{scheduled-workflow-run} | Get scheduled workflow run |
| [**workflow_scheduled_list**](WorkflowApi.md#workflow_scheduled_list) | **GET** /api/v1/tenants/{tenant}/workflows/scheduled | Get scheduled workflow runs |
| [**workflow_update**](WorkflowApi.md#workflow_update) | **PATCH** /api/v1/workflows/{workflow} | Update workflow |
| [**workflow_version_get**](WorkflowApi.md#workflow_version_get) | **GET** /api/v1/workflows/{workflow}/versions | Get workflow version |


## cron_workflow_list

> <CronWorkflowsList> cron_workflow_list(tenant, opts)

Get cron job workflows

Get all cron job workflow triggers for a tenant

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

api_instance = HatchetSdkRest::WorkflowApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
opts = {
  offset: 789, # Integer | The number to skip
  limit: 789, # Integer | The number to limit by
  workflow_id: '38400000-8cf0-11bd-b23e-10b96e4ef00d', # String | The workflow id to get runs for.
  workflow_name: 'workflow_name_example', # String | The workflow name to get runs for.
  cron_name: 'cron_name_example', # String | The cron name to get runs for.
  additional_metadata: ['inner_example'], # Array<String> | A list of metadata key value pairs to filter by
  order_by_field: HatchetSdkRest::CronWorkflowsOrderByField::NAME, # CronWorkflowsOrderByField | The order by field
  order_by_direction: HatchetSdkRest::WorkflowRunOrderByDirection::ASC # WorkflowRunOrderByDirection | The order by direction
}

begin
  # Get cron job workflows
  result = api_instance.cron_workflow_list(tenant, opts)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowApi->cron_workflow_list: #{e}"
end
```

#### Using the cron_workflow_list_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<CronWorkflowsList>, Integer, Hash)> cron_workflow_list_with_http_info(tenant, opts)

```ruby
begin
  # Get cron job workflows
  data, status_code, headers = api_instance.cron_workflow_list_with_http_info(tenant, opts)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <CronWorkflowsList>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowApi->cron_workflow_list_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **offset** | **Integer** | The number to skip | [optional] |
| **limit** | **Integer** | The number to limit by | [optional] |
| **workflow_id** | **String** | The workflow id to get runs for. | [optional] |
| **workflow_name** | **String** | The workflow name to get runs for. | [optional] |
| **cron_name** | **String** | The cron name to get runs for. | [optional] |
| **additional_metadata** | [**Array&lt;String&gt;**](String.md) | A list of metadata key value pairs to filter by | [optional] |
| **order_by_field** | [**CronWorkflowsOrderByField**](.md) | The order by field | [optional] |
| **order_by_direction** | [**WorkflowRunOrderByDirection**](.md) | The order by direction | [optional] |

### Return type

[**CronWorkflowsList**](CronWorkflowsList.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## tenant_get_queue_metrics

> <TenantQueueMetrics> tenant_get_queue_metrics(tenant, opts)

Get workflow metrics

Get the queue metrics for the tenant

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

api_instance = HatchetSdkRest::WorkflowApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
opts = {
  workflows: ['inner_example'], # Array<String> | A list of workflow IDs to filter by
  additional_metadata: ['inner_example'] # Array<String> | A list of metadata key value pairs to filter by
}

begin
  # Get workflow metrics
  result = api_instance.tenant_get_queue_metrics(tenant, opts)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowApi->tenant_get_queue_metrics: #{e}"
end
```

#### Using the tenant_get_queue_metrics_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<TenantQueueMetrics>, Integer, Hash)> tenant_get_queue_metrics_with_http_info(tenant, opts)

```ruby
begin
  # Get workflow metrics
  data, status_code, headers = api_instance.tenant_get_queue_metrics_with_http_info(tenant, opts)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <TenantQueueMetrics>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowApi->tenant_get_queue_metrics_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **workflows** | [**Array&lt;String&gt;**](String.md) | A list of workflow IDs to filter by | [optional] |
| **additional_metadata** | [**Array&lt;String&gt;**](String.md) | A list of metadata key value pairs to filter by | [optional] |

### Return type

[**TenantQueueMetrics**](TenantQueueMetrics.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## workflow_cron_delete

> workflow_cron_delete(tenant, cron_workflow)

Delete cron job workflow run

Delete a cron job workflow run for a tenant

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

api_instance = HatchetSdkRest::WorkflowApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
cron_workflow = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The cron job id

begin
  # Delete cron job workflow run
  api_instance.workflow_cron_delete(tenant, cron_workflow)
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowApi->workflow_cron_delete: #{e}"
end
```

#### Using the workflow_cron_delete_with_http_info variant

This returns an Array which contains the response data (`nil` in this case), status code and headers.

> <Array(nil, Integer, Hash)> workflow_cron_delete_with_http_info(tenant, cron_workflow)

```ruby
begin
  # Delete cron job workflow run
  data, status_code, headers = api_instance.workflow_cron_delete_with_http_info(tenant, cron_workflow)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => nil
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowApi->workflow_cron_delete_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **cron_workflow** | **String** | The cron job id |  |

### Return type

nil (empty response body)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## workflow_cron_get

> <CronWorkflows> workflow_cron_get(tenant, cron_workflow)

Get cron job workflow run

Get a cron job workflow run for a tenant

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

api_instance = HatchetSdkRest::WorkflowApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
cron_workflow = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The cron job id

begin
  # Get cron job workflow run
  result = api_instance.workflow_cron_get(tenant, cron_workflow)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowApi->workflow_cron_get: #{e}"
end
```

#### Using the workflow_cron_get_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<CronWorkflows>, Integer, Hash)> workflow_cron_get_with_http_info(tenant, cron_workflow)

```ruby
begin
  # Get cron job workflow run
  data, status_code, headers = api_instance.workflow_cron_get_with_http_info(tenant, cron_workflow)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <CronWorkflows>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowApi->workflow_cron_get_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **cron_workflow** | **String** | The cron job id |  |

### Return type

[**CronWorkflows**](CronWorkflows.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## workflow_delete

> workflow_delete(workflow)

Delete workflow

Delete a workflow for a tenant

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

api_instance = HatchetSdkRest::WorkflowApi.new
workflow = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The workflow id

begin
  # Delete workflow
  api_instance.workflow_delete(workflow)
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowApi->workflow_delete: #{e}"
end
```

#### Using the workflow_delete_with_http_info variant

This returns an Array which contains the response data (`nil` in this case), status code and headers.

> <Array(nil, Integer, Hash)> workflow_delete_with_http_info(workflow)

```ruby
begin
  # Delete workflow
  data, status_code, headers = api_instance.workflow_delete_with_http_info(workflow)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => nil
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowApi->workflow_delete_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **workflow** | **String** | The workflow id |  |

### Return type

nil (empty response body)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## workflow_get

> <Workflow> workflow_get(workflow)

Get workflow

Get a workflow for a tenant

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

api_instance = HatchetSdkRest::WorkflowApi.new
workflow = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The workflow id

begin
  # Get workflow
  result = api_instance.workflow_get(workflow)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowApi->workflow_get: #{e}"
end
```

#### Using the workflow_get_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<Workflow>, Integer, Hash)> workflow_get_with_http_info(workflow)

```ruby
begin
  # Get workflow
  data, status_code, headers = api_instance.workflow_get_with_http_info(workflow)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <Workflow>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowApi->workflow_get_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **workflow** | **String** | The workflow id |  |

### Return type

[**Workflow**](Workflow.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## workflow_get_metrics

> <WorkflowMetrics> workflow_get_metrics(workflow, opts)

Get workflow metrics

Get the metrics for a workflow version

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

api_instance = HatchetSdkRest::WorkflowApi.new
workflow = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The workflow id
opts = {
  status: HatchetSdkRest::WorkflowRunStatus::PENDING, # WorkflowRunStatus | A status of workflow run statuses to filter by
  group_key: 'group_key_example' # String | A group key to filter metrics by
}

begin
  # Get workflow metrics
  result = api_instance.workflow_get_metrics(workflow, opts)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowApi->workflow_get_metrics: #{e}"
end
```

#### Using the workflow_get_metrics_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<WorkflowMetrics>, Integer, Hash)> workflow_get_metrics_with_http_info(workflow, opts)

```ruby
begin
  # Get workflow metrics
  data, status_code, headers = api_instance.workflow_get_metrics_with_http_info(workflow, opts)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <WorkflowMetrics>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowApi->workflow_get_metrics_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **workflow** | **String** | The workflow id |  |
| **status** | [**WorkflowRunStatus**](.md) | A status of workflow run statuses to filter by | [optional] |
| **group_key** | **String** | A group key to filter metrics by | [optional] |

### Return type

[**WorkflowMetrics**](WorkflowMetrics.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## workflow_get_workers_count

> <WorkflowWorkersCount> workflow_get_workers_count(tenant, workflow)

Get workflow worker count

Get a count of the workers available for workflow

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

api_instance = HatchetSdkRest::WorkflowApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
workflow = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The workflow id

begin
  # Get workflow worker count
  result = api_instance.workflow_get_workers_count(tenant, workflow)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowApi->workflow_get_workers_count: #{e}"
end
```

#### Using the workflow_get_workers_count_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<WorkflowWorkersCount>, Integer, Hash)> workflow_get_workers_count_with_http_info(tenant, workflow)

```ruby
begin
  # Get workflow worker count
  data, status_code, headers = api_instance.workflow_get_workers_count_with_http_info(tenant, workflow)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <WorkflowWorkersCount>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowApi->workflow_get_workers_count_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **workflow** | **String** | The workflow id |  |

### Return type

[**WorkflowWorkersCount**](WorkflowWorkersCount.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## workflow_list

> <WorkflowList> workflow_list(tenant, opts)

Get workflows

Get all workflows for a tenant

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

api_instance = HatchetSdkRest::WorkflowApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
opts = {
  offset: 56, # Integer | The number to skip
  limit: 56, # Integer | The number to limit by
  name: 'name_example' # String | Search by name
}

begin
  # Get workflows
  result = api_instance.workflow_list(tenant, opts)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowApi->workflow_list: #{e}"
end
```

#### Using the workflow_list_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<WorkflowList>, Integer, Hash)> workflow_list_with_http_info(tenant, opts)

```ruby
begin
  # Get workflows
  data, status_code, headers = api_instance.workflow_list_with_http_info(tenant, opts)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <WorkflowList>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowApi->workflow_list_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **offset** | **Integer** | The number to skip | [optional][default to 0] |
| **limit** | **Integer** | The number to limit by | [optional][default to 50] |
| **name** | **String** | Search by name | [optional] |

### Return type

[**WorkflowList**](WorkflowList.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## workflow_run_get

> <WorkflowRun> workflow_run_get(tenant, workflow_run)

Get workflow run

Get a workflow run for a tenant

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

api_instance = HatchetSdkRest::WorkflowApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
workflow_run = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The workflow run id

begin
  # Get workflow run
  result = api_instance.workflow_run_get(tenant, workflow_run)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowApi->workflow_run_get: #{e}"
end
```

#### Using the workflow_run_get_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<WorkflowRun>, Integer, Hash)> workflow_run_get_with_http_info(tenant, workflow_run)

```ruby
begin
  # Get workflow run
  data, status_code, headers = api_instance.workflow_run_get_with_http_info(tenant, workflow_run)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <WorkflowRun>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowApi->workflow_run_get_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **workflow_run** | **String** | The workflow run id |  |

### Return type

[**WorkflowRun**](WorkflowRun.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## workflow_run_get_metrics

> <WorkflowRunsMetrics> workflow_run_get_metrics(tenant, opts)

Get workflow runs metrics

Get a summary of  workflow run metrics for a tenant

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

api_instance = HatchetSdkRest::WorkflowApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
opts = {
  event_id: '38400000-8cf0-11bd-b23e-10b96e4ef00d', # String | The event id to get runs for.
  workflow_id: '38400000-8cf0-11bd-b23e-10b96e4ef00d', # String | The workflow id to get runs for.
  parent_workflow_run_id: '38400000-8cf0-11bd-b23e-10b96e4ef00d', # String | The parent workflow run id
  parent_step_run_id: '38400000-8cf0-11bd-b23e-10b96e4ef00d', # String | The parent step run id
  additional_metadata: ['inner_example'], # Array<String> | A list of metadata key value pairs to filter by
  created_after: Time.parse('2021-01-01T00:00:00Z'), # Time | The time after the workflow run was created
  created_before: Time.parse('2021-01-01T00:00:00Z') # Time | The time before the workflow run was created
}

begin
  # Get workflow runs metrics
  result = api_instance.workflow_run_get_metrics(tenant, opts)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowApi->workflow_run_get_metrics: #{e}"
end
```

#### Using the workflow_run_get_metrics_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<WorkflowRunsMetrics>, Integer, Hash)> workflow_run_get_metrics_with_http_info(tenant, opts)

```ruby
begin
  # Get workflow runs metrics
  data, status_code, headers = api_instance.workflow_run_get_metrics_with_http_info(tenant, opts)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <WorkflowRunsMetrics>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowApi->workflow_run_get_metrics_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **event_id** | **String** | The event id to get runs for. | [optional] |
| **workflow_id** | **String** | The workflow id to get runs for. | [optional] |
| **parent_workflow_run_id** | **String** | The parent workflow run id | [optional] |
| **parent_step_run_id** | **String** | The parent step run id | [optional] |
| **additional_metadata** | [**Array&lt;String&gt;**](String.md) | A list of metadata key value pairs to filter by | [optional] |
| **created_after** | **Time** | The time after the workflow run was created | [optional] |
| **created_before** | **Time** | The time before the workflow run was created | [optional] |

### Return type

[**WorkflowRunsMetrics**](WorkflowRunsMetrics.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## workflow_run_get_shape

> <WorkflowRunShape> workflow_run_get_shape(tenant, workflow_run)

Get workflow run

Get a workflow run for a tenant

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

api_instance = HatchetSdkRest::WorkflowApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
workflow_run = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The workflow run id

begin
  # Get workflow run
  result = api_instance.workflow_run_get_shape(tenant, workflow_run)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowApi->workflow_run_get_shape: #{e}"
end
```

#### Using the workflow_run_get_shape_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<WorkflowRunShape>, Integer, Hash)> workflow_run_get_shape_with_http_info(tenant, workflow_run)

```ruby
begin
  # Get workflow run
  data, status_code, headers = api_instance.workflow_run_get_shape_with_http_info(tenant, workflow_run)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <WorkflowRunShape>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowApi->workflow_run_get_shape_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **workflow_run** | **String** | The workflow run id |  |

### Return type

[**WorkflowRunShape**](WorkflowRunShape.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## workflow_run_list

> <WorkflowRunList> workflow_run_list(tenant, opts)

Get workflow runs

Get all workflow runs for a tenant

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

api_instance = HatchetSdkRest::WorkflowApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
opts = {
  offset: 789, # Integer | The number to skip
  limit: 789, # Integer | The number to limit by
  event_id: '38400000-8cf0-11bd-b23e-10b96e4ef00d', # String | The event id to get runs for.
  workflow_id: '38400000-8cf0-11bd-b23e-10b96e4ef00d', # String | The workflow id to get runs for.
  parent_workflow_run_id: '38400000-8cf0-11bd-b23e-10b96e4ef00d', # String | The parent workflow run id
  parent_step_run_id: '38400000-8cf0-11bd-b23e-10b96e4ef00d', # String | The parent step run id
  statuses: [HatchetSdkRest::WorkflowRunStatus::PENDING], # Array<WorkflowRunStatus> | A list of workflow run statuses to filter by
  kinds: [HatchetSdkRest::WorkflowKind::FUNCTION], # Array<WorkflowKind> | A list of workflow kinds to filter by
  additional_metadata: ['inner_example'], # Array<String> | A list of metadata key value pairs to filter by
  created_after: Time.parse('2021-01-01T00:00:00Z'), # Time | The time after the workflow run was created
  created_before: Time.parse('2021-01-01T00:00:00Z'), # Time | The time before the workflow run was created
  finished_after: Time.parse('2021-01-01T00:00:00Z'), # Time | The time after the workflow run was finished
  finished_before: Time.parse('2021-01-01T00:00:00Z'), # Time | The time before the workflow run was finished
  order_by_field: HatchetSdkRest::WorkflowRunOrderByField::CREATED_AT, # WorkflowRunOrderByField | The order by field
  order_by_direction: HatchetSdkRest::WorkflowRunOrderByDirection::ASC # WorkflowRunOrderByDirection | The order by direction
}

begin
  # Get workflow runs
  result = api_instance.workflow_run_list(tenant, opts)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowApi->workflow_run_list: #{e}"
end
```

#### Using the workflow_run_list_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<WorkflowRunList>, Integer, Hash)> workflow_run_list_with_http_info(tenant, opts)

```ruby
begin
  # Get workflow runs
  data, status_code, headers = api_instance.workflow_run_list_with_http_info(tenant, opts)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <WorkflowRunList>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowApi->workflow_run_list_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **offset** | **Integer** | The number to skip | [optional] |
| **limit** | **Integer** | The number to limit by | [optional] |
| **event_id** | **String** | The event id to get runs for. | [optional] |
| **workflow_id** | **String** | The workflow id to get runs for. | [optional] |
| **parent_workflow_run_id** | **String** | The parent workflow run id | [optional] |
| **parent_step_run_id** | **String** | The parent step run id | [optional] |
| **statuses** | [**Array&lt;WorkflowRunStatus&gt;**](WorkflowRunStatus.md) | A list of workflow run statuses to filter by | [optional] |
| **kinds** | [**Array&lt;WorkflowKind&gt;**](WorkflowKind.md) | A list of workflow kinds to filter by | [optional] |
| **additional_metadata** | [**Array&lt;String&gt;**](String.md) | A list of metadata key value pairs to filter by | [optional] |
| **created_after** | **Time** | The time after the workflow run was created | [optional] |
| **created_before** | **Time** | The time before the workflow run was created | [optional] |
| **finished_after** | **Time** | The time after the workflow run was finished | [optional] |
| **finished_before** | **Time** | The time before the workflow run was finished | [optional] |
| **order_by_field** | [**WorkflowRunOrderByField**](.md) | The order by field | [optional] |
| **order_by_direction** | [**WorkflowRunOrderByDirection**](.md) | The order by direction | [optional] |

### Return type

[**WorkflowRunList**](WorkflowRunList.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## workflow_scheduled_delete

> workflow_scheduled_delete(tenant, scheduled_workflow_run)

Delete scheduled workflow run

Delete a scheduled workflow run for a tenant

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

api_instance = HatchetSdkRest::WorkflowApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
scheduled_workflow_run = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The scheduled workflow id

begin
  # Delete scheduled workflow run
  api_instance.workflow_scheduled_delete(tenant, scheduled_workflow_run)
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowApi->workflow_scheduled_delete: #{e}"
end
```

#### Using the workflow_scheduled_delete_with_http_info variant

This returns an Array which contains the response data (`nil` in this case), status code and headers.

> <Array(nil, Integer, Hash)> workflow_scheduled_delete_with_http_info(tenant, scheduled_workflow_run)

```ruby
begin
  # Delete scheduled workflow run
  data, status_code, headers = api_instance.workflow_scheduled_delete_with_http_info(tenant, scheduled_workflow_run)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => nil
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowApi->workflow_scheduled_delete_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **scheduled_workflow_run** | **String** | The scheduled workflow id |  |

### Return type

nil (empty response body)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## workflow_scheduled_get

> <ScheduledWorkflows> workflow_scheduled_get(tenant, scheduled_workflow_run)

Get scheduled workflow run

Get a scheduled workflow run for a tenant

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

api_instance = HatchetSdkRest::WorkflowApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
scheduled_workflow_run = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The scheduled workflow id

begin
  # Get scheduled workflow run
  result = api_instance.workflow_scheduled_get(tenant, scheduled_workflow_run)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowApi->workflow_scheduled_get: #{e}"
end
```

#### Using the workflow_scheduled_get_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<ScheduledWorkflows>, Integer, Hash)> workflow_scheduled_get_with_http_info(tenant, scheduled_workflow_run)

```ruby
begin
  # Get scheduled workflow run
  data, status_code, headers = api_instance.workflow_scheduled_get_with_http_info(tenant, scheduled_workflow_run)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <ScheduledWorkflows>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowApi->workflow_scheduled_get_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **scheduled_workflow_run** | **String** | The scheduled workflow id |  |

### Return type

[**ScheduledWorkflows**](ScheduledWorkflows.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## workflow_scheduled_list

> <ScheduledWorkflowsList> workflow_scheduled_list(tenant, opts)

Get scheduled workflow runs

Get all scheduled workflow runs for a tenant

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

api_instance = HatchetSdkRest::WorkflowApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
opts = {
  offset: 789, # Integer | The number to skip
  limit: 789, # Integer | The number to limit by
  order_by_field: HatchetSdkRest::ScheduledWorkflowsOrderByField::TRIGGER_AT, # ScheduledWorkflowsOrderByField | The order by field
  order_by_direction: HatchetSdkRest::WorkflowRunOrderByDirection::ASC, # WorkflowRunOrderByDirection | The order by direction
  workflow_id: '38400000-8cf0-11bd-b23e-10b96e4ef00d', # String | The workflow id to get runs for.
  parent_workflow_run_id: '38400000-8cf0-11bd-b23e-10b96e4ef00d', # String | The parent workflow run id
  parent_step_run_id: '38400000-8cf0-11bd-b23e-10b96e4ef00d', # String | The parent step run id
  additional_metadata: ['inner_example'], # Array<String> | A list of metadata key value pairs to filter by
  statuses: [HatchetSdkRest::ScheduledRunStatus::PENDING] # Array<ScheduledRunStatus> | A list of scheduled run statuses to filter by
}

begin
  # Get scheduled workflow runs
  result = api_instance.workflow_scheduled_list(tenant, opts)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowApi->workflow_scheduled_list: #{e}"
end
```

#### Using the workflow_scheduled_list_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<ScheduledWorkflowsList>, Integer, Hash)> workflow_scheduled_list_with_http_info(tenant, opts)

```ruby
begin
  # Get scheduled workflow runs
  data, status_code, headers = api_instance.workflow_scheduled_list_with_http_info(tenant, opts)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <ScheduledWorkflowsList>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowApi->workflow_scheduled_list_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **offset** | **Integer** | The number to skip | [optional] |
| **limit** | **Integer** | The number to limit by | [optional] |
| **order_by_field** | [**ScheduledWorkflowsOrderByField**](.md) | The order by field | [optional] |
| **order_by_direction** | [**WorkflowRunOrderByDirection**](.md) | The order by direction | [optional] |
| **workflow_id** | **String** | The workflow id to get runs for. | [optional] |
| **parent_workflow_run_id** | **String** | The parent workflow run id | [optional] |
| **parent_step_run_id** | **String** | The parent step run id | [optional] |
| **additional_metadata** | [**Array&lt;String&gt;**](String.md) | A list of metadata key value pairs to filter by | [optional] |
| **statuses** | [**Array&lt;ScheduledRunStatus&gt;**](ScheduledRunStatus.md) | A list of scheduled run statuses to filter by | [optional] |

### Return type

[**ScheduledWorkflowsList**](ScheduledWorkflowsList.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## workflow_update

> <Workflow> workflow_update(workflow, workflow_update_request)

Update workflow

Update a workflow for a tenant

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

api_instance = HatchetSdkRest::WorkflowApi.new
workflow = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The workflow id
workflow_update_request = HatchetSdkRest::WorkflowUpdateRequest.new # WorkflowUpdateRequest | The input to update the workflow

begin
  # Update workflow
  result = api_instance.workflow_update(workflow, workflow_update_request)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowApi->workflow_update: #{e}"
end
```

#### Using the workflow_update_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<Workflow>, Integer, Hash)> workflow_update_with_http_info(workflow, workflow_update_request)

```ruby
begin
  # Update workflow
  data, status_code, headers = api_instance.workflow_update_with_http_info(workflow, workflow_update_request)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <Workflow>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowApi->workflow_update_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **workflow** | **String** | The workflow id |  |
| **workflow_update_request** | [**WorkflowUpdateRequest**](WorkflowUpdateRequest.md) | The input to update the workflow |  |

### Return type

[**Workflow**](Workflow.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json


## workflow_version_get

> <WorkflowVersion> workflow_version_get(workflow, opts)

Get workflow version

Get a workflow version for a tenant

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

api_instance = HatchetSdkRest::WorkflowApi.new
workflow = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The workflow id
opts = {
  version: '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The workflow version. If not supplied, the latest version is fetched.
}

begin
  # Get workflow version
  result = api_instance.workflow_version_get(workflow, opts)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowApi->workflow_version_get: #{e}"
end
```

#### Using the workflow_version_get_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<WorkflowVersion>, Integer, Hash)> workflow_version_get_with_http_info(workflow, opts)

```ruby
begin
  # Get workflow version
  data, status_code, headers = api_instance.workflow_version_get_with_http_info(workflow, opts)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <WorkflowVersion>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowApi->workflow_version_get_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **workflow** | **String** | The workflow id |  |
| **version** | **String** | The workflow version. If not supplied, the latest version is fetched. | [optional] |

### Return type

[**WorkflowVersion**](WorkflowVersion.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

