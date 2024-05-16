# WorkflowVersionDefinition


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**raw_definition** | **str** | The raw YAML definition of the workflow. |

## Example

```python
from hatchet_sdk.clients.rest.models.workflow_version_definition import WorkflowVersionDefinition

# TODO update the JSON string below
json = "{}"
# create an instance of WorkflowVersionDefinition from a JSON string
workflow_version_definition_instance = WorkflowVersionDefinition.from_json(json)
# print the JSON string representation of the object
print WorkflowVersionDefinition.to_json()

# convert the object into a dict
workflow_version_definition_dict = workflow_version_definition_instance.to_dict()
# create an instance of WorkflowVersionDefinition from a dict
workflow_version_definition_form_dict = workflow_version_definition.from_dict(workflow_version_definition_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
