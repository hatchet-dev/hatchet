# WorkflowRunTriggeredBy


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**metadata** | [**APIResourceMeta**](APIResourceMeta.md) |  |
**parent_id** | **str** |  |
**event_id** | **str** |  | [optional]
**event** | [**Event**](Event.md) |  | [optional]
**cron_parent_id** | **str** |  | [optional]
**cron_schedule** | **str** |  | [optional]

## Example

```python
from hatchet_sdk.clients.rest.models.workflow_run_triggered_by import WorkflowRunTriggeredBy

# TODO update the JSON string below
json = "{}"
# create an instance of WorkflowRunTriggeredBy from a JSON string
workflow_run_triggered_by_instance = WorkflowRunTriggeredBy.from_json(json)
# print the JSON string representation of the object
print WorkflowRunTriggeredBy.to_json()

# convert the object into a dict
workflow_run_triggered_by_dict = workflow_run_triggered_by_instance.to_dict()
# create an instance of WorkflowRunTriggeredBy from a dict
workflow_run_triggered_by_form_dict = workflow_run_triggered_by.from_dict(workflow_run_triggered_by_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
