# PaginationResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**current_page** | **int** | the current page | [optional]
**next_page** | **int** | the next page | [optional]
**num_pages** | **int** | the total number of pages for listing | [optional]

## Example

```python
from hatchet_sdk.clients.rest.models.pagination_response import PaginationResponse

# TODO update the JSON string below
json = "{}"
# create an instance of PaginationResponse from a JSON string
pagination_response_instance = PaginationResponse.from_json(json)
# print the JSON string representation of the object
print PaginationResponse.to_json()

# convert the object into a dict
pagination_response_dict = pagination_response_instance.to_dict()
# create an instance of PaginationResponse from a dict
pagination_response_form_dict = pagination_response.from_dict(pagination_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
