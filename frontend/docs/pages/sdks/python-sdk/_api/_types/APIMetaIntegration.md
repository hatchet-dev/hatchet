# APIMetaIntegration


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **str** | the name of the integration |
**enabled** | **bool** | whether this integration is enabled on the instance |

## Example

```python
from hatchet_sdk.clients.rest.models.api_meta_integration import APIMetaIntegration

# TODO update the JSON string below
json = "{}"
# create an instance of APIMetaIntegration from a JSON string
api_meta_integration_instance = APIMetaIntegration.from_json(json)
# print the JSON string representation of the object
print APIMetaIntegration.to_json()

# convert the object into a dict
api_meta_integration_dict = api_meta_integration_instance.to_dict()
# create an instance of APIMetaIntegration from a dict
api_meta_integration_form_dict = api_meta_integration.from_dict(api_meta_integration_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
