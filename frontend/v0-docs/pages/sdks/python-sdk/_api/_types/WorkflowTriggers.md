# WorkflowTriggers


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**metadata** | [**APIResourceMeta**](APIResourceMeta.md) |  | [optional]
**workflow_version_id** | **str** |  | [optional]
**tenant_id** | **str** |  | [optional]
**events** | [**List[WorkflowTriggerEventRef]**](WorkflowTriggerEventRef.md) |  | [optional]
**crons** | [**List[WorkflowTriggerCronRef]**](WorkflowTriggerCronRef.md) |  | [optional]

## Example

```python
from hatchet_sdk.clients.rest.models.workflow_triggers import WorkflowTriggers

# TODO update the JSON string below
json = "{}"
# create an instance of WorkflowTriggers from a JSON string
workflow_triggers_instance = WorkflowTriggers.from_json(json)
# print the JSON string representation of the object
print WorkflowTriggers.to_json()

# convert the object into a dict
workflow_triggers_dict = workflow_triggers_instance.to_dict()
# create an instance of WorkflowTriggers from a dict
workflow_triggers_form_dict = workflow_triggers.from_dict(workflow_triggers_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
