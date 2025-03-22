# APIMetaAuth


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**schemes** | **List[str]** | the supported types of authentication | [optional]

## Example

```python
from hatchet_sdk.clients.rest.models.api_meta_auth import APIMetaAuth

# TODO update the JSON string below
json = "{}"
# create an instance of APIMetaAuth from a JSON string
api_meta_auth_instance = APIMetaAuth.from_json(json)
# print the JSON string representation of the object
print APIMetaAuth.to_json()

# convert the object into a dict
api_meta_auth_dict = api_meta_auth_instance.to_dict()
# create an instance of APIMetaAuth from a dict
api_meta_auth_form_dict = api_meta_auth.from_dict(api_meta_auth_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
