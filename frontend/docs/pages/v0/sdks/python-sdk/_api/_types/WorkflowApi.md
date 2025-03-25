# hatchet_sdk.clients.rest.WorkflowApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**step_run_get_diff**](WorkflowApi.md#step_run_get_diff) | **GET** /api/v1/step-runs/{step-run}/diff | Get diff
[**step_run_update_create_pr**](WorkflowApi.md#step_run_update_create_pr) | **POST** /api/v1/step-runs/{step-run}/create-pr | Create pull request
[**workflow_get**](WorkflowApi.md#workflow_get) | **GET** /api/v1/workflows/{workflow} | Get workflow
[**workflow_list**](WorkflowApi.md#workflow_list) | **GET** /api/v1/tenants/{tenant}/workflows | Get workflows
[**workflow_run_get**](WorkflowApi.md#workflow_run_get) | **GET** /api/v1/tenants/{tenant}/workflow-runs/{workflow-run} | Get workflow run
[**workflow_run_list**](WorkflowApi.md#workflow_run_list) | **GET** /api/v1/tenants/{tenant}/workflows/runs | Get workflow runs
[**workflow_run_list_pull_requests**](WorkflowApi.md#workflow_run_list_pull_requests) | **GET** /api/v1/tenants/{tenant}/workflow-runs/{workflow-run}/prs | List pull requests
[**workflow_update_link_github**](WorkflowApi.md#workflow_update_link_github) | **POST** /api/v1/workflows/{workflow}/link-github | Link github repository
[**workflow_version_get**](WorkflowApi.md#workflow_version_get) | **GET** /api/v1/workflows/{workflow}/versions | Get workflow version
[**workflow_version_get_definition**](WorkflowApi.md#workflow_version_get_definition) | **GET** /api/v1/workflows/{workflow}/versions/definition | Get workflow version definition


# **step_run_get_diff**
> GetStepRunDiffResponse step_run_get_diff(step_run)

Get diff

Get the diff for a step run between the most recent run and the first run.

### Example

* Api Key Authentication (cookieAuth):
* Bearer Authentication (bearerAuth):

```python
import hatchet_sdk.clients.rest
from hatchet_sdk.clients.rest.models.get_step_run_diff_response import GetStepRunDiffResponse
from hatchet_sdk.clients.rest.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = hatchet_sdk.clients.rest.Configuration(
    host = "http://localhost"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: cookieAuth
configuration.api_key['cookieAuth'] = os.environ["API_KEY"]

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['cookieAuth'] = 'Bearer'

# Configure Bearer authorization: bearerAuth
configuration = hatchet_sdk.clients.rest.Configuration(
    access_token = os.environ["BEARER_TOKEN"]
)

# Enter a context with an instance of the API client
with hatchet_sdk.clients.rest.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = hatchet_sdk.clients.rest.WorkflowApi(api_client)
    step_run = 'step_run_example' # str | The step run id

    try:
        # Get diff
        api_response = api_instance.step_run_get_diff(step_run)
        print("The response of WorkflowApi->step_run_get_diff:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling WorkflowApi->step_run_get_diff: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **step_run** | **str**| The step run id |

### Return type

[**GetStepRunDiffResponse**](GetStepRunDiffResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Successfully retrieved the diff |  -  |
**400** | A malformed or bad request |  -  |
**403** | Forbidden |  -  |
**404** | Not found |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **step_run_update_create_pr**
> CreatePullRequestFromStepRun step_run_update_create_pr(step_run, create_pull_request_from_step_run)

Create pull request

Create a pull request for a workflow

### Example

* Api Key Authentication (cookieAuth):
* Bearer Authentication (bearerAuth):

```python
import hatchet_sdk.clients.rest
from hatchet_sdk.clients.rest.models.create_pull_request_from_step_run import CreatePullRequestFromStepRun
from hatchet_sdk.clients.rest.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = hatchet_sdk.clients.rest.Configuration(
    host = "http://localhost"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: cookieAuth
configuration.api_key['cookieAuth'] = os.environ["API_KEY"]

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['cookieAuth'] = 'Bearer'

# Configure Bearer authorization: bearerAuth
configuration = hatchet_sdk.clients.rest.Configuration(
    access_token = os.environ["BEARER_TOKEN"]
)

# Enter a context with an instance of the API client
with hatchet_sdk.clients.rest.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = hatchet_sdk.clients.rest.WorkflowApi(api_client)
    step_run = 'step_run_example' # str | The step run id
    create_pull_request_from_step_run = hatchet_sdk.clients.rest.CreatePullRequestFromStepRun() # CreatePullRequestFromStepRun | The input to create a pull request

    try:
        # Create pull request
        api_response = api_instance.step_run_update_create_pr(step_run, create_pull_request_from_step_run)
        print("The response of WorkflowApi->step_run_update_create_pr:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling WorkflowApi->step_run_update_create_pr: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **step_run** | **str**| The step run id |
 **create_pull_request_from_step_run** | [**CreatePullRequestFromStepRun**](CreatePullRequestFromStepRun.md)| The input to create a pull request |

### Return type

[**CreatePullRequestFromStepRun**](CreatePullRequestFromStepRun.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Successfully created the pull request |  -  |
**400** | A malformed or bad request |  -  |
**403** | Forbidden |  -  |
**404** | Not found |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **workflow_get**
> Workflow workflow_get(workflow)

Get workflow

Get a workflow for a tenant

### Example

* Api Key Authentication (cookieAuth):
* Bearer Authentication (bearerAuth):

```python
import hatchet_sdk.clients.rest
from hatchet_sdk.clients.rest.models.workflow import Workflow
from hatchet_sdk.clients.rest.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = hatchet_sdk.clients.rest.Configuration(
    host = "http://localhost"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: cookieAuth
configuration.api_key['cookieAuth'] = os.environ["API_KEY"]

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['cookieAuth'] = 'Bearer'

# Configure Bearer authorization: bearerAuth
configuration = hatchet_sdk.clients.rest.Configuration(
    access_token = os.environ["BEARER_TOKEN"]
)

# Enter a context with an instance of the API client
with hatchet_sdk.clients.rest.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = hatchet_sdk.clients.rest.WorkflowApi(api_client)
    workflow = 'workflow_example' # str | The workflow id

    try:
        # Get workflow
        api_response = api_instance.workflow_get(workflow)
        print("The response of WorkflowApi->workflow_get:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling WorkflowApi->workflow_get: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **workflow** | **str**| The workflow id |

### Return type

[**Workflow**](Workflow.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Successfully retrieved the workflow |  -  |
**400** | A malformed or bad request |  -  |
**403** | Forbidden |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **workflow_list**
> WorkflowList workflow_list(tenant)

Get workflows

Get all workflows for a tenant

### Example

* Api Key Authentication (cookieAuth):
* Bearer Authentication (bearerAuth):

```python
import hatchet_sdk.clients.rest
from hatchet_sdk.clients.rest.models.workflow_list import WorkflowList
from hatchet_sdk.clients.rest.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = hatchet_sdk.clients.rest.Configuration(
    host = "http://localhost"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: cookieAuth
configuration.api_key['cookieAuth'] = os.environ["API_KEY"]

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['cookieAuth'] = 'Bearer'

# Configure Bearer authorization: bearerAuth
configuration = hatchet_sdk.clients.rest.Configuration(
    access_token = os.environ["BEARER_TOKEN"]
)

# Enter a context with an instance of the API client
with hatchet_sdk.clients.rest.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = hatchet_sdk.clients.rest.WorkflowApi(api_client)
    tenant = 'tenant_example' # str | The tenant id

    try:
        # Get workflows
        api_response = api_instance.workflow_list(tenant)
        print("The response of WorkflowApi->workflow_list:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling WorkflowApi->workflow_list: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **tenant** | **str**| The tenant id |

### Return type

[**WorkflowList**](WorkflowList.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Successfully retrieved the workflows |  -  |
**400** | A malformed or bad request |  -  |
**403** | Forbidden |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **workflow_run_get**
> WorkflowRun workflow_run_get(tenant, workflow_run)

Get workflow run

Get a workflow run for a tenant

### Example

* Api Key Authentication (cookieAuth):
* Bearer Authentication (bearerAuth):

```python
import hatchet_sdk.clients.rest
from hatchet_sdk.clients.rest.models.workflow_run import WorkflowRun
from hatchet_sdk.clients.rest.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = hatchet_sdk.clients.rest.Configuration(
    host = "http://localhost"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: cookieAuth
configuration.api_key['cookieAuth'] = os.environ["API_KEY"]

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['cookieAuth'] = 'Bearer'

# Configure Bearer authorization: bearerAuth
configuration = hatchet_sdk.clients.rest.Configuration(
    access_token = os.environ["BEARER_TOKEN"]
)

# Enter a context with an instance of the API client
with hatchet_sdk.clients.rest.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = hatchet_sdk.clients.rest.WorkflowApi(api_client)
    tenant = 'tenant_example' # str | The tenant id
    workflow_run = 'workflow_run_example' # str | The workflow run id

    try:
        # Get workflow run
        api_response = api_instance.workflow_run_get(tenant, workflow_run)
        print("The response of WorkflowApi->workflow_run_get:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling WorkflowApi->workflow_run_get: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **tenant** | **str**| The tenant id |
 **workflow_run** | **str**| The workflow run id |

### Return type

[**WorkflowRun**](WorkflowRun.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Successfully retrieved the workflow run |  -  |
**400** | A malformed or bad request |  -  |
**403** | Forbidden |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **workflow_run_list**
> WorkflowRunList workflow_run_list(tenant, offset=offset, limit=limit, event_id=event_id, workflow_id=workflow_id)

Get workflow runs

Get all workflow runs for a tenant

### Example

* Api Key Authentication (cookieAuth):
* Bearer Authentication (bearerAuth):

```python
import hatchet_sdk.clients.rest
from hatchet_sdk.clients.rest.models.workflow_run_list import WorkflowRunList
from hatchet_sdk.clients.rest.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = hatchet_sdk.clients.rest.Configuration(
    host = "http://localhost"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: cookieAuth
configuration.api_key['cookieAuth'] = os.environ["API_KEY"]

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['cookieAuth'] = 'Bearer'

# Configure Bearer authorization: bearerAuth
configuration = hatchet_sdk.clients.rest.Configuration(
    access_token = os.environ["BEARER_TOKEN"]
)

# Enter a context with an instance of the API client
with hatchet_sdk.clients.rest.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = hatchet_sdk.clients.rest.WorkflowApi(api_client)
    tenant = 'tenant_example' # str | The tenant id
    offset = 56 # int | The number to skip (optional)
    limit = 56 # int | The number to limit by (optional)
    event_id = 'event_id_example' # str | The event id to get runs for. (optional)
    workflow_id = 'workflow_id_example' # str | The workflow id to get runs for. (optional)

    try:
        # Get workflow runs
        api_response = api_instance.workflow_run_list(tenant, offset=offset, limit=limit, event_id=event_id, workflow_id=workflow_id)
        print("The response of WorkflowApi->workflow_run_list:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling WorkflowApi->workflow_run_list: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **tenant** | **str**| The tenant id |
 **offset** | **int**| The number to skip | [optional]
 **limit** | **int**| The number to limit by | [optional]
 **event_id** | **str**| The event id to get runs for. | [optional]
 **workflow_id** | **str**| The workflow id to get runs for. | [optional]

### Return type

[**WorkflowRunList**](WorkflowRunList.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Successfully retrieved the workflow runs |  -  |
**400** | A malformed or bad request |  -  |
**403** | Forbidden |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **workflow_run_list_pull_requests**
> ListPullRequestsResponse workflow_run_list_pull_requests(tenant, workflow_run, state=state)

List pull requests

List all pull requests for a workflow run

### Example

* Api Key Authentication (cookieAuth):
* Bearer Authentication (bearerAuth):

```python
import hatchet_sdk.clients.rest
from hatchet_sdk.clients.rest.models.list_pull_requests_response import ListPullRequestsResponse
from hatchet_sdk.clients.rest.models.pull_request_state import PullRequestState
from hatchet_sdk.clients.rest.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = hatchet_sdk.clients.rest.Configuration(
    host = "http://localhost"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: cookieAuth
configuration.api_key['cookieAuth'] = os.environ["API_KEY"]

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['cookieAuth'] = 'Bearer'

# Configure Bearer authorization: bearerAuth
configuration = hatchet_sdk.clients.rest.Configuration(
    access_token = os.environ["BEARER_TOKEN"]
)

# Enter a context with an instance of the API client
with hatchet_sdk.clients.rest.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = hatchet_sdk.clients.rest.WorkflowApi(api_client)
    tenant = 'tenant_example' # str | The tenant id
    workflow_run = 'workflow_run_example' # str | The workflow run id
    state = hatchet_sdk.clients.rest.PullRequestState() # PullRequestState | The pull request state (optional)

    try:
        # List pull requests
        api_response = api_instance.workflow_run_list_pull_requests(tenant, workflow_run, state=state)
        print("The response of WorkflowApi->workflow_run_list_pull_requests:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling WorkflowApi->workflow_run_list_pull_requests: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **tenant** | **str**| The tenant id |
 **workflow_run** | **str**| The workflow run id |
 **state** | [**PullRequestState**](.md)| The pull request state | [optional]

### Return type

[**ListPullRequestsResponse**](ListPullRequestsResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Successfully retrieved the list of pull requests |  -  |
**400** | A malformed or bad request |  -  |
**403** | Forbidden |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **workflow_update_link_github**
> Workflow workflow_update_link_github(workflow, link_github_repository_request)

Link github repository

Link a github repository to a workflow

### Example

* Api Key Authentication (cookieAuth):
* Bearer Authentication (bearerAuth):

```python
import hatchet_sdk.clients.rest
from hatchet_sdk.clients.rest.models.link_github_repository_request import LinkGithubRepositoryRequest
from hatchet_sdk.clients.rest.models.workflow import Workflow
from hatchet_sdk.clients.rest.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = hatchet_sdk.clients.rest.Configuration(
    host = "http://localhost"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: cookieAuth
configuration.api_key['cookieAuth'] = os.environ["API_KEY"]

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['cookieAuth'] = 'Bearer'

# Configure Bearer authorization: bearerAuth
configuration = hatchet_sdk.clients.rest.Configuration(
    access_token = os.environ["BEARER_TOKEN"]
)

# Enter a context with an instance of the API client
with hatchet_sdk.clients.rest.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = hatchet_sdk.clients.rest.WorkflowApi(api_client)
    workflow = 'workflow_example' # str | The workflow id
    link_github_repository_request = hatchet_sdk.clients.rest.LinkGithubRepositoryRequest() # LinkGithubRepositoryRequest | The input to link a github repository

    try:
        # Link github repository
        api_response = api_instance.workflow_update_link_github(workflow, link_github_repository_request)
        print("The response of WorkflowApi->workflow_update_link_github:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling WorkflowApi->workflow_update_link_github: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **workflow** | **str**| The workflow id |
 **link_github_repository_request** | [**LinkGithubRepositoryRequest**](LinkGithubRepositoryRequest.md)| The input to link a github repository |

### Return type

[**Workflow**](Workflow.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Successfully linked the github repository |  -  |
**400** | A malformed or bad request |  -  |
**403** | Forbidden |  -  |
**404** | Not found |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **workflow_version_get**
> WorkflowVersion workflow_version_get(workflow, version=version)

Get workflow version

Get a workflow version for a tenant

### Example

* Api Key Authentication (cookieAuth):
* Bearer Authentication (bearerAuth):

```python
import hatchet_sdk.clients.rest
from hatchet_sdk.clients.rest.models.workflow_version import WorkflowVersion
from hatchet_sdk.clients.rest.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = hatchet_sdk.clients.rest.Configuration(
    host = "http://localhost"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: cookieAuth
configuration.api_key['cookieAuth'] = os.environ["API_KEY"]

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['cookieAuth'] = 'Bearer'

# Configure Bearer authorization: bearerAuth
configuration = hatchet_sdk.clients.rest.Configuration(
    access_token = os.environ["BEARER_TOKEN"]
)

# Enter a context with an instance of the API client
with hatchet_sdk.clients.rest.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = hatchet_sdk.clients.rest.WorkflowApi(api_client)
    workflow = 'workflow_example' # str | The workflow id
    version = 'version_example' # str | The workflow version. If not supplied, the latest version is fetched. (optional)

    try:
        # Get workflow version
        api_response = api_instance.workflow_version_get(workflow, version=version)
        print("The response of WorkflowApi->workflow_version_get:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling WorkflowApi->workflow_version_get: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **workflow** | **str**| The workflow id |
 **version** | **str**| The workflow version. If not supplied, the latest version is fetched. | [optional]

### Return type

[**WorkflowVersion**](WorkflowVersion.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Successfully retrieved the workflow version |  -  |
**400** | A malformed or bad request |  -  |
**403** | Forbidden |  -  |
**404** | Not found |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **workflow_version_get_definition**
> WorkflowVersionDefinition workflow_version_get_definition(workflow, version=version)

Get workflow version definition

Get a workflow version definition for a tenant

### Example

* Api Key Authentication (cookieAuth):
* Bearer Authentication (bearerAuth):

```python
import hatchet_sdk.clients.rest
from hatchet_sdk.clients.rest.models.workflow_version_definition import WorkflowVersionDefinition
from hatchet_sdk.clients.rest.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = hatchet_sdk.clients.rest.Configuration(
    host = "http://localhost"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: cookieAuth
configuration.api_key['cookieAuth'] = os.environ["API_KEY"]

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['cookieAuth'] = 'Bearer'

# Configure Bearer authorization: bearerAuth
configuration = hatchet_sdk.clients.rest.Configuration(
    access_token = os.environ["BEARER_TOKEN"]
)

# Enter a context with an instance of the API client
with hatchet_sdk.clients.rest.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = hatchet_sdk.clients.rest.WorkflowApi(api_client)
    workflow = 'workflow_example' # str | The workflow id
    version = 'version_example' # str | The workflow version. If not supplied, the latest version is fetched. (optional)

    try:
        # Get workflow version definition
        api_response = api_instance.workflow_version_get_definition(workflow, version=version)
        print("The response of WorkflowApi->workflow_version_get_definition:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling WorkflowApi->workflow_version_get_definition: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **workflow** | **str**| The workflow id |
 **version** | **str**| The workflow version. If not supplied, the latest version is fetched. | [optional]

### Return type

[**WorkflowVersionDefinition**](WorkflowVersionDefinition.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Successfully retrieved the workflow version definition |  -  |
**400** | A malformed or bad request |  -  |
**403** | Forbidden |  -  |
**404** | Not found |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)
