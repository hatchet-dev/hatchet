# APIResourceMeta


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **str** | the id of this resource, in UUID format |
**created_at** | **datetime** | the time that this resource was created |
**updated_at** | **datetime** | the time that this resource was last updated |

## Example

```python
from hatchet_sdk.clients.rest.models.api_resource_meta import APIResourceMeta

# TODO update the JSON string below
json = "{}"
# create an instance of APIResourceMeta from a JSON string
api_resource_meta_instance = APIResourceMeta.from_json(json)
# print the JSON string representation of the object
print APIResourceMeta.to_json()

# convert the object into a dict
api_resource_meta_dict = api_resource_meta_instance.to_dict()
# create an instance of APIResourceMeta from a dict
api_resource_meta_form_dict = api_resource_meta.from_dict(api_resource_meta_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
