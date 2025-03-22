# WorkflowVersionMeta


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**metadata** | [**APIResourceMeta**](APIResourceMeta.md) |  |
**version** | **str** | The version of the workflow. |
**order** | **int** |  |
**workflow_id** | **str** |  |
**workflow** | [**Workflow**](Workflow.md) |  | [optional]

## Example

```python
from hatchet_sdk.clients.rest.models.workflow_version_meta import WorkflowVersionMeta

# TODO update the JSON string below
json = "{}"
# create an instance of WorkflowVersionMeta from a JSON string
workflow_version_meta_instance = WorkflowVersionMeta.from_json(json)
# print the JSON string representation of the object
print WorkflowVersionMeta.to_json()

# convert the object into a dict
workflow_version_meta_dict = workflow_version_meta_instance.to_dict()
# create an instance of WorkflowVersionMeta from a dict
workflow_version_meta_form_dict = workflow_version_meta.from_dict(workflow_version_meta_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
