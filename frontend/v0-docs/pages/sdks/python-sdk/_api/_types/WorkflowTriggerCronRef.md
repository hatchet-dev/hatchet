# WorkflowTriggerCronRef


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**parent_id** | **str** |  | [optional]
**cron** | **str** |  | [optional]

## Example

```python
from hatchet_sdk.clients.rest.models.workflow_trigger_cron_ref import WorkflowTriggerCronRef

# TODO update the JSON string below
json = "{}"
# create an instance of WorkflowTriggerCronRef from a JSON string
workflow_trigger_cron_ref_instance = WorkflowTriggerCronRef.from_json(json)
# print the JSON string representation of the object
print WorkflowTriggerCronRef.to_json()

# convert the object into a dict
workflow_trigger_cron_ref_dict = workflow_trigger_cron_ref_instance.to_dict()
# create an instance of WorkflowTriggerCronRef from a dict
workflow_trigger_cron_ref_form_dict = workflow_trigger_cron_ref.from_dict(workflow_trigger_cron_ref_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
