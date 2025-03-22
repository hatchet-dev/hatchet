# WorkflowRunList


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**rows** | [**List[WorkflowRun]**](WorkflowRun.md) |  | [optional]
**pagination** | [**PaginationResponse**](PaginationResponse.md) |  | [optional]

## Example

```python
from hatchet_sdk.clients.rest.models.workflow_run_list import WorkflowRunList

# TODO update the JSON string below
json = "{}"
# create an instance of WorkflowRunList from a JSON string
workflow_run_list_instance = WorkflowRunList.from_json(json)
# print the JSON string representation of the object
print WorkflowRunList.to_json()

# convert the object into a dict
workflow_run_list_dict = workflow_run_list_instance.to_dict()
# create an instance of WorkflowRunList from a dict
workflow_run_list_form_dict = workflow_run_list.from_dict(workflow_run_list_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
