# Workflow


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**metadata** | [**APIResourceMeta**](APIResourceMeta.md) |  |
**name** | **str** | The name of the workflow. |
**description** | **str** | The description of the workflow. | [optional]
**versions** | [**List[WorkflowVersionMeta]**](WorkflowVersionMeta.md) |  | [optional]
**tags** | [**List[WorkflowTag]**](WorkflowTag.md) | The tags of the workflow. | [optional]
**last_run** | [**WorkflowRun**](WorkflowRun.md) |  | [optional]
**jobs** | [**List[Job]**](Job.md) | The jobs of the workflow. | [optional]
**deployment** | [**WorkflowDeploymentConfig**](WorkflowDeploymentConfig.md) |  | [optional]

## Example

```python
from hatchet_sdk.clients.rest.models.workflow import Workflow

# TODO update the JSON string below
json = "{}"
# create an instance of Workflow from a JSON string
workflow_instance = Workflow.from_json(json)
# print the JSON string representation of the object
print Workflow.to_json()

# convert the object into a dict
workflow_dict = workflow_instance.to_dict()
# create an instance of Workflow from a dict
workflow_form_dict = workflow.from_dict(workflow_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
