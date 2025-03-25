# APIMeta


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**auth** | [**APIMetaAuth**](APIMetaAuth.md) |  | [optional]

## Example

```python
from hatchet_sdk.clients.rest.models.api_meta import APIMeta

# TODO update the JSON string below
json = "{}"
# create an instance of APIMeta from a JSON string
api_meta_instance = APIMeta.from_json(json)
# print the JSON string representation of the object
print APIMeta.to_json()

# convert the object into a dict
api_meta_dict = api_meta_instance.to_dict()
# create an instance of APIMeta from a dict
api_meta_form_dict = api_meta.from_dict(api_meta_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
