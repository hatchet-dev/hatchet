# WorkflowDeploymentConfig


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**metadata** | [**APIResourceMeta**](APIResourceMeta.md) |  |
**git_repo_name** | **str** | The repository name. |
**git_repo_owner** | **str** | The repository owner. |
**git_repo_branch** | **str** | The repository branch. |
**github_app_installation** | [**GithubAppInstallation**](GithubAppInstallation.md) | The Github App installation. | [optional]
**github_app_installation_id** | **str** | The id of the Github App installation. |

## Example

```python
from hatchet_sdk.clients.rest.models.workflow_deployment_config import WorkflowDeploymentConfig

# TODO update the JSON string below
json = "{}"
# create an instance of WorkflowDeploymentConfig from a JSON string
workflow_deployment_config_instance = WorkflowDeploymentConfig.from_json(json)
# print the JSON string representation of the object
print WorkflowDeploymentConfig.to_json()

# convert the object into a dict
workflow_deployment_config_dict = workflow_deployment_config_instance.to_dict()
# create an instance of WorkflowDeploymentConfig from a dict
workflow_deployment_config_form_dict = workflow_deployment_config.from_dict(workflow_deployment_config_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
