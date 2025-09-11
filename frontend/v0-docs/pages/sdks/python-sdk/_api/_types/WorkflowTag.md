# WorkflowTag


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **str** | The name of the workflow. |
**color** | **str** | The description of the workflow. |

## Example

```python
from hatchet_sdk.clients.rest.models.workflow_tag import WorkflowTag

# TODO update the JSON string below
json = "{}"
# create an instance of WorkflowTag from a JSON string
workflow_tag_instance = WorkflowTag.from_json(json)
# print the JSON string representation of the object
print WorkflowTag.to_json()

# convert the object into a dict
workflow_tag_dict = workflow_tag_instance.to_dict()
# create an instance of WorkflowTag from a dict
workflow_tag_form_dict = workflow_tag.from_dict(workflow_tag_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
