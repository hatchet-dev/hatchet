# APIError


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**code** | **int** | a custom Hatchet error code | [optional]
**field** | **str** | the field that this error is associated with, if applicable | [optional]
**description** | **str** | a description for this error |
**docs_link** | **str** | a link to the documentation for this error, if it exists | [optional]

## Example

```python
from hatchet_sdk.clients.rest.models.api_error import APIError

# TODO update the JSON string below
json = "{}"
# create an instance of APIError from a JSON string
api_error_instance = APIError.from_json(json)
# print the JSON string representation of the object
print APIError.to_json()

# convert the object into a dict
api_error_dict = api_error_instance.to_dict()
# create an instance of APIError from a dict
api_error_form_dict = api_error.from_dict(api_error_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
