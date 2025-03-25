# WorkflowList


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**metadata** | [**APIResourceMeta**](APIResourceMeta.md) |  | [optional]
**rows** | [**List[Workflow]**](Workflow.md) |  | [optional]
**pagination** | [**PaginationResponse**](PaginationResponse.md) |  | [optional]

## Example

```python
from hatchet_sdk.clients.rest.models.workflow_list import WorkflowList

# TODO update the JSON string below
json = "{}"
# create an instance of WorkflowList from a JSON string
workflow_list_instance = WorkflowList.from_json(json)
# print the JSON string representation of the object
print WorkflowList.to_json()

# convert the object into a dict
workflow_list_dict = workflow_list_instance.to_dict()
# create an instance of WorkflowList from a dict
workflow_list_form_dict = workflow_list.from_dict(workflow_list_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
