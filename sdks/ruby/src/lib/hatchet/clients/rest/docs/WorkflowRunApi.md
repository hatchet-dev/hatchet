# HatchetSdkRest::WorkflowRunApi

All URIs are relative to *http://localhost*

| Method | HTTP request | Description |
| ------ | ------------ | ----------- |
| [**cron_workflow_trigger_create**](WorkflowRunApi.md#cron_workflow_trigger_create) | **POST** /api/v1/tenants/{tenant}/workflows/{workflow}/crons | Create cron job workflow trigger |
| [**scheduled_workflow_run_create**](WorkflowRunApi.md#scheduled_workflow_run_create) | **POST** /api/v1/tenants/{tenant}/workflows/{workflow}/scheduled | Trigger workflow run |
| [**workflow_run_cancel**](WorkflowRunApi.md#workflow_run_cancel) | **POST** /api/v1/tenants/{tenant}/workflows/cancel | Cancel workflow runs |
| [**workflow_run_create**](WorkflowRunApi.md#workflow_run_create) | **POST** /api/v1/workflows/{workflow}/trigger | Trigger workflow run |
| [**workflow_run_get_input**](WorkflowRunApi.md#workflow_run_get_input) | **GET** /api/v1/tenants/{tenant}/workflow-runs/{workflow-run}/input | Get workflow run input |
| [**workflow_run_update_replay**](WorkflowRunApi.md#workflow_run_update_replay) | **POST** /api/v1/tenants/{tenant}/workflow-runs/replay | Replay workflow runs |


## cron_workflow_trigger_create

> <CronWorkflows> cron_workflow_trigger_create(tenant, workflow, create_cron_workflow_trigger_request)

Create cron job workflow trigger

Create a new cron job workflow trigger for a tenant

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

api_instance = HatchetSdkRest::WorkflowRunApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
workflow = 'workflow_example' # String | The workflow name
create_cron_workflow_trigger_request = HatchetSdkRest::CreateCronWorkflowTriggerRequest.new({input: 3.56, additional_metadata: 3.56, cron_name: 'cron_name_example', cron_expression: 'cron_expression_example'}) # CreateCronWorkflowTriggerRequest | The input to the cron job workflow trigger

begin
  # Create cron job workflow trigger
  result = api_instance.cron_workflow_trigger_create(tenant, workflow, create_cron_workflow_trigger_request)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowRunApi->cron_workflow_trigger_create: #{e}"
end
```

#### Using the cron_workflow_trigger_create_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<CronWorkflows>, Integer, Hash)> cron_workflow_trigger_create_with_http_info(tenant, workflow, create_cron_workflow_trigger_request)

```ruby
begin
  # Create cron job workflow trigger
  data, status_code, headers = api_instance.cron_workflow_trigger_create_with_http_info(tenant, workflow, create_cron_workflow_trigger_request)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <CronWorkflows>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowRunApi->cron_workflow_trigger_create_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **workflow** | **String** | The workflow name |  |
| **create_cron_workflow_trigger_request** | [**CreateCronWorkflowTriggerRequest**](CreateCronWorkflowTriggerRequest.md) | The input to the cron job workflow trigger |  |

### Return type

[**CronWorkflows**](CronWorkflows.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json


## scheduled_workflow_run_create

> <ScheduledWorkflows> scheduled_workflow_run_create(tenant, workflow, schedule_workflow_run_request)

Trigger workflow run

Schedule a new workflow run for a tenant

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

api_instance = HatchetSdkRest::WorkflowRunApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
workflow = 'workflow_example' # String | The workflow name
schedule_workflow_run_request = HatchetSdkRest::ScheduleWorkflowRunRequest.new({input: 3.56, additional_metadata: 3.56, trigger_at: Time.now}) # ScheduleWorkflowRunRequest | The input to the scheduled workflow run

begin
  # Trigger workflow run
  result = api_instance.scheduled_workflow_run_create(tenant, workflow, schedule_workflow_run_request)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowRunApi->scheduled_workflow_run_create: #{e}"
end
```

#### Using the scheduled_workflow_run_create_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<ScheduledWorkflows>, Integer, Hash)> scheduled_workflow_run_create_with_http_info(tenant, workflow, schedule_workflow_run_request)

```ruby
begin
  # Trigger workflow run
  data, status_code, headers = api_instance.scheduled_workflow_run_create_with_http_info(tenant, workflow, schedule_workflow_run_request)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <ScheduledWorkflows>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowRunApi->scheduled_workflow_run_create_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **workflow** | **String** | The workflow name |  |
| **schedule_workflow_run_request** | [**ScheduleWorkflowRunRequest**](ScheduleWorkflowRunRequest.md) | The input to the scheduled workflow run |  |

### Return type

[**ScheduledWorkflows**](ScheduledWorkflows.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json


## workflow_run_cancel

> <EventUpdateCancel200Response> workflow_run_cancel(tenant, workflow_runs_cancel_request)

Cancel workflow runs

Cancel a batch of workflow runs

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

api_instance = HatchetSdkRest::WorkflowRunApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
workflow_runs_cancel_request = HatchetSdkRest::WorkflowRunsCancelRequest.new({workflow_run_ids: ['workflow_run_ids_example']}) # WorkflowRunsCancelRequest | The input to cancel the workflow runs

begin
  # Cancel workflow runs
  result = api_instance.workflow_run_cancel(tenant, workflow_runs_cancel_request)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowRunApi->workflow_run_cancel: #{e}"
end
```

#### Using the workflow_run_cancel_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<EventUpdateCancel200Response>, Integer, Hash)> workflow_run_cancel_with_http_info(tenant, workflow_runs_cancel_request)

```ruby
begin
  # Cancel workflow runs
  data, status_code, headers = api_instance.workflow_run_cancel_with_http_info(tenant, workflow_runs_cancel_request)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <EventUpdateCancel200Response>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowRunApi->workflow_run_cancel_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **workflow_runs_cancel_request** | [**WorkflowRunsCancelRequest**](WorkflowRunsCancelRequest.md) | The input to cancel the workflow runs |  |

### Return type

[**EventUpdateCancel200Response**](EventUpdateCancel200Response.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json


## workflow_run_create

> <WorkflowRun> workflow_run_create(workflow, trigger_workflow_run_request, opts)

Trigger workflow run

Trigger a new workflow run for a tenant

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

api_instance = HatchetSdkRest::WorkflowRunApi.new
workflow = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The workflow id
trigger_workflow_run_request = HatchetSdkRest::TriggerWorkflowRunRequest.new({input: 3.56}) # TriggerWorkflowRunRequest | The input to the workflow run
opts = {
  version: '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The workflow version. If not supplied, the latest version is fetched.
}

begin
  # Trigger workflow run
  result = api_instance.workflow_run_create(workflow, trigger_workflow_run_request, opts)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowRunApi->workflow_run_create: #{e}"
end
```

#### Using the workflow_run_create_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<WorkflowRun>, Integer, Hash)> workflow_run_create_with_http_info(workflow, trigger_workflow_run_request, opts)

```ruby
begin
  # Trigger workflow run
  data, status_code, headers = api_instance.workflow_run_create_with_http_info(workflow, trigger_workflow_run_request, opts)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <WorkflowRun>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowRunApi->workflow_run_create_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **workflow** | **String** | The workflow id |  |
| **trigger_workflow_run_request** | [**TriggerWorkflowRunRequest**](TriggerWorkflowRunRequest.md) | The input to the workflow run |  |
| **version** | **String** | The workflow version. If not supplied, the latest version is fetched. | [optional] |

### Return type

[**WorkflowRun**](WorkflowRun.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json


## workflow_run_get_input

> Hash&lt;String, Object&gt; workflow_run_get_input(tenant, workflow_run)

Get workflow run input

Get the input for a workflow run.

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

api_instance = HatchetSdkRest::WorkflowRunApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
workflow_run = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The workflow run id

begin
  # Get workflow run input
  result = api_instance.workflow_run_get_input(tenant, workflow_run)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowRunApi->workflow_run_get_input: #{e}"
end
```

#### Using the workflow_run_get_input_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(Hash&lt;String, Object&gt;, Integer, Hash)> workflow_run_get_input_with_http_info(tenant, workflow_run)

```ruby
begin
  # Get workflow run input
  data, status_code, headers = api_instance.workflow_run_get_input_with_http_info(tenant, workflow_run)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => Hash&lt;String, Object&gt;
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowRunApi->workflow_run_get_input_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **workflow_run** | **String** | The workflow run id |  |

### Return type

**Hash&lt;String, Object&gt;**

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## workflow_run_update_replay

> <ReplayWorkflowRunsResponse> workflow_run_update_replay(tenant, replay_workflow_runs_request)

Replay workflow runs

Replays a list of workflow runs.

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

api_instance = HatchetSdkRest::WorkflowRunApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
replay_workflow_runs_request = HatchetSdkRest::ReplayWorkflowRunsRequest.new({workflow_run_ids: ['bb214807-246e-43a5-a25d-41761d1cff9e']}) # ReplayWorkflowRunsRequest | The workflow run ids to replay

begin
  # Replay workflow runs
  result = api_instance.workflow_run_update_replay(tenant, replay_workflow_runs_request)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowRunApi->workflow_run_update_replay: #{e}"
end
```

#### Using the workflow_run_update_replay_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<ReplayWorkflowRunsResponse>, Integer, Hash)> workflow_run_update_replay_with_http_info(tenant, replay_workflow_runs_request)

```ruby
begin
  # Replay workflow runs
  data, status_code, headers = api_instance.workflow_run_update_replay_with_http_info(tenant, replay_workflow_runs_request)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <ReplayWorkflowRunsResponse>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling WorkflowRunApi->workflow_run_update_replay_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **replay_workflow_runs_request** | [**ReplayWorkflowRunsRequest**](ReplayWorkflowRunsRequest.md) | The workflow run ids to replay |  |

### Return type

[**ReplayWorkflowRunsResponse**](ReplayWorkflowRunsResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

