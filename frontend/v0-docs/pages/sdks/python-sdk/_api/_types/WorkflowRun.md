# WorkflowRun


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**metadata** | [**APIResourceMeta**](APIResourceMeta.md) |  |
**tenant_id** | **str** |  |
**workflow_version_id** | **str** |  |
**workflow_version** | [**WorkflowVersion**](WorkflowVersion.md) |  | [optional]
**status** | [**WorkflowRunStatus**](WorkflowRunStatus.md) |  |
**display_name** | **str** |  | [optional]
**job_runs** | [**List[JobRun]**](JobRun.md) |  | [optional]
**triggered_by** | [**WorkflowRunTriggeredBy**](WorkflowRunTriggeredBy.md) |  |
**input** | **Dict[str, object]** |  | [optional]
**error** | **str** |  | [optional]
**started_at** | **datetime** |  | [optional]
**finished_at** | **datetime** |  | [optional]

## Example

```python
from hatchet_sdk.clients.rest.models.workflow_run import WorkflowRun

# TODO update the JSON string below
json = "{}"
# create an instance of WorkflowRun from a JSON string
workflow_run_instance = WorkflowRun.from_json(json)
# print the JSON string representation of the object
print WorkflowRun.to_json()

# convert the object into a dict
workflow_run_dict = workflow_run_instance.to_dict()
# create an instance of WorkflowRun from a dict
workflow_run_form_dict = workflow_run.from_dict(workflow_run_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
