# hatchet_sdk.clients.rest.WorkflowRunApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**workflow_run_create**](WorkflowRunApi.md#workflow_run_create) | **POST** /api/v1/workflows/{workflow}/trigger | Trigger workflow run


# **workflow_run_create**
> WorkflowRun workflow_run_create(workflow, trigger_workflow_run_request, version=version)

Trigger workflow run

Trigger a new workflow run for a tenant

### Example

* Api Key Authentication (cookieAuth):
* Bearer Authentication (bearerAuth):

```python
import hatchet_sdk.clients.rest
from hatchet_sdk.clients.rest.models.trigger_workflow_run_request import TriggerWorkflowRunRequest
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
    api_instance = hatchet_sdk.clients.rest.WorkflowRunApi(api_client)
    workflow = 'workflow_example' # str | The workflow id
    trigger_workflow_run_request = hatchet_sdk.clients.rest.TriggerWorkflowRunRequest() # TriggerWorkflowRunRequest | The input to the workflow run
    version = 'version_example' # str | The workflow version. If not supplied, the latest version is fetched. (optional)

    try:
        # Trigger workflow run
        api_response = api_instance.workflow_run_create(workflow, trigger_workflow_run_request, version=version)
        print("The response of WorkflowRunApi->workflow_run_create:\n")
        print(api_response)
    except Exception as e:
        print("Exception when calling WorkflowRunApi->workflow_run_create: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **workflow** | **str**| The workflow id |
 **trigger_workflow_run_request** | [**TriggerWorkflowRunRequest**](TriggerWorkflowRunRequest.md)| The input to the workflow run |
 **version** | **str**| The workflow version. If not supplied, the latest version is fetched. | [optional]

### Return type

[**WorkflowRun**](WorkflowRun.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Successfully created the workflow run |  -  |
**400** | A malformed or bad request |  -  |
**403** | Forbidden |  -  |
**404** | Not found |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)
