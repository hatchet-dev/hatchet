# WorkflowVersion


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**metadata** | [**APIResourceMeta**](APIResourceMeta.md) |  |
**version** | **str** | The version of the workflow. |
**order** | **int** |  |
**workflow_id** | **str** |  |
**workflow** | [**Workflow**](Workflow.md) |  | [optional]
**triggers** | [**WorkflowTriggers**](WorkflowTriggers.md) |  | [optional]
**jobs** | [**List[Job]**](Job.md) |  | [optional]

## Example

```python
from hatchet_sdk.clients.rest.models.workflow_version import WorkflowVersion

# TODO update the JSON string below
json = "{}"
# create an instance of WorkflowVersion from a JSON string
workflow_version_instance = WorkflowVersion.from_json(json)
# print the JSON string representation of the object
print WorkflowVersion.to_json()

# convert the object into a dict
workflow_version_dict = workflow_version_instance.to_dict()
# create an instance of WorkflowVersion from a dict
workflow_version_form_dict = workflow_version.from_dict(workflow_version_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
