# HatchetSdkRest::EventApi

All URIs are relative to *http://localhost*

| Method | HTTP request | Description |
| ------ | ------------ | ----------- |
| [**event_create**](EventApi.md#event_create) | **POST** /api/v1/tenants/{tenant}/events | Create event |
| [**event_create_bulk**](EventApi.md#event_create_bulk) | **POST** /api/v1/tenants/{tenant}/events/bulk | Bulk Create events |
| [**event_data_get**](EventApi.md#event_data_get) | **GET** /api/v1/events/{event}/data | Get event data |
| [**event_get**](EventApi.md#event_get) | **GET** /api/v1/events/{event} | Get event data |
| [**event_key_list**](EventApi.md#event_key_list) | **GET** /api/v1/tenants/{tenant}/events/keys | List event keys |
| [**event_list**](EventApi.md#event_list) | **GET** /api/v1/tenants/{tenant}/events | List events |
| [**event_update_cancel**](EventApi.md#event_update_cancel) | **POST** /api/v1/tenants/{tenant}/events/cancel | Replay events |
| [**event_update_replay**](EventApi.md#event_update_replay) | **POST** /api/v1/tenants/{tenant}/events/replay | Replay events |
| [**v1_event_key_list**](EventApi.md#v1_event_key_list) | **GET** /api/v1/stable/tenants/{tenant}/events/keys | List event keys |
| [**v1_event_list**](EventApi.md#v1_event_list) | **GET** /api/v1/stable/tenants/{tenant}/events | List events |


## event_create

> <Event> event_create(tenant, create_event_request)

Create event

Creates a new event.

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'
# setup authorization
HatchetSdkRest.configure do |config|
  # Configure API key authorization: cookieAuth
  config.api_key['hatchet'] = 'YOUR API KEY'
  # Uncomment the following line to set a prefix for the API key, e.g. 'Bearer' (defaults to nil)
  # config.api_key_prefix['hatchet'] = 'Bearer'

  # Configure Bearer authorization: bearerAuth
  config.access_token = 'YOUR_BEARER_TOKEN'
end

api_instance = HatchetSdkRest::EventApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
create_event_request = HatchetSdkRest::CreateEventRequest.new({key: 'key_example', data: 3.56}) # CreateEventRequest | The event to create

begin
  # Create event
  result = api_instance.event_create(tenant, create_event_request)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling EventApi->event_create: #{e}"
end
```

#### Using the event_create_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<Event>, Integer, Hash)> event_create_with_http_info(tenant, create_event_request)

```ruby
begin
  # Create event
  data, status_code, headers = api_instance.event_create_with_http_info(tenant, create_event_request)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <Event>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling EventApi->event_create_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **create_event_request** | [**CreateEventRequest**](CreateEventRequest.md) | The event to create |  |

### Return type

[**Event**](Event.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json


## event_create_bulk

> <Events> event_create_bulk(tenant, bulk_create_event_request)

Bulk Create events

Bulk creates new events.

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'
# setup authorization
HatchetSdkRest.configure do |config|
  # Configure API key authorization: cookieAuth
  config.api_key['hatchet'] = 'YOUR API KEY'
  # Uncomment the following line to set a prefix for the API key, e.g. 'Bearer' (defaults to nil)
  # config.api_key_prefix['hatchet'] = 'Bearer'

  # Configure Bearer authorization: bearerAuth
  config.access_token = 'YOUR_BEARER_TOKEN'
end

api_instance = HatchetSdkRest::EventApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
bulk_create_event_request = HatchetSdkRest::BulkCreateEventRequest.new({events: [HatchetSdkRest::CreateEventRequest.new({key: 'key_example', data: 3.56})]}) # BulkCreateEventRequest | The events to create

begin
  # Bulk Create events
  result = api_instance.event_create_bulk(tenant, bulk_create_event_request)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling EventApi->event_create_bulk: #{e}"
end
```

#### Using the event_create_bulk_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<Events>, Integer, Hash)> event_create_bulk_with_http_info(tenant, bulk_create_event_request)

```ruby
begin
  # Bulk Create events
  data, status_code, headers = api_instance.event_create_bulk_with_http_info(tenant, bulk_create_event_request)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <Events>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling EventApi->event_create_bulk_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **bulk_create_event_request** | [**BulkCreateEventRequest**](BulkCreateEventRequest.md) | The events to create |  |

### Return type

[**Events**](Events.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json


## event_data_get

> <EventData> event_data_get(event)

Get event data

Get the data for an event.

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'
# setup authorization
HatchetSdkRest.configure do |config|
  # Configure API key authorization: cookieAuth
  config.api_key['hatchet'] = 'YOUR API KEY'
  # Uncomment the following line to set a prefix for the API key, e.g. 'Bearer' (defaults to nil)
  # config.api_key_prefix['hatchet'] = 'Bearer'

  # Configure Bearer authorization: bearerAuth
  config.access_token = 'YOUR_BEARER_TOKEN'
end

api_instance = HatchetSdkRest::EventApi.new
event = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The event id

begin
  # Get event data
  result = api_instance.event_data_get(event)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling EventApi->event_data_get: #{e}"
end
```

#### Using the event_data_get_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<EventData>, Integer, Hash)> event_data_get_with_http_info(event)

```ruby
begin
  # Get event data
  data, status_code, headers = api_instance.event_data_get_with_http_info(event)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <EventData>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling EventApi->event_data_get_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **event** | **String** | The event id |  |

### Return type

[**EventData**](EventData.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## event_get

> <Event> event_get(event)

Get event data

Get an event.

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'
# setup authorization
HatchetSdkRest.configure do |config|
  # Configure API key authorization: cookieAuth
  config.api_key['hatchet'] = 'YOUR API KEY'
  # Uncomment the following line to set a prefix for the API key, e.g. 'Bearer' (defaults to nil)
  # config.api_key_prefix['hatchet'] = 'Bearer'

  # Configure Bearer authorization: bearerAuth
  config.access_token = 'YOUR_BEARER_TOKEN'
end

api_instance = HatchetSdkRest::EventApi.new
event = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The event id

begin
  # Get event data
  result = api_instance.event_get(event)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling EventApi->event_get: #{e}"
end
```

#### Using the event_get_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<Event>, Integer, Hash)> event_get_with_http_info(event)

```ruby
begin
  # Get event data
  data, status_code, headers = api_instance.event_get_with_http_info(event)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <Event>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling EventApi->event_get_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **event** | **String** | The event id |  |

### Return type

[**Event**](Event.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## event_key_list

> <EventKeyList> event_key_list(tenant)

List event keys

Lists all event keys for a tenant.

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'
# setup authorization
HatchetSdkRest.configure do |config|
  # Configure API key authorization: cookieAuth
  config.api_key['hatchet'] = 'YOUR API KEY'
  # Uncomment the following line to set a prefix for the API key, e.g. 'Bearer' (defaults to nil)
  # config.api_key_prefix['hatchet'] = 'Bearer'

  # Configure Bearer authorization: bearerAuth
  config.access_token = 'YOUR_BEARER_TOKEN'
end

api_instance = HatchetSdkRest::EventApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id

begin
  # List event keys
  result = api_instance.event_key_list(tenant)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling EventApi->event_key_list: #{e}"
end
```

#### Using the event_key_list_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<EventKeyList>, Integer, Hash)> event_key_list_with_http_info(tenant)

```ruby
begin
  # List event keys
  data, status_code, headers = api_instance.event_key_list_with_http_info(tenant)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <EventKeyList>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling EventApi->event_key_list_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |

### Return type

[**EventKeyList**](EventKeyList.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## event_list

> <EventList> event_list(tenant, opts)

List events

Lists all events for a tenant.

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'
# setup authorization
HatchetSdkRest.configure do |config|
  # Configure API key authorization: cookieAuth
  config.api_key['hatchet'] = 'YOUR API KEY'
  # Uncomment the following line to set a prefix for the API key, e.g. 'Bearer' (defaults to nil)
  # config.api_key_prefix['hatchet'] = 'Bearer'

  # Configure Bearer authorization: bearerAuth
  config.access_token = 'YOUR_BEARER_TOKEN'
end

api_instance = HatchetSdkRest::EventApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
opts = {
  offset: 789, # Integer | The number to skip
  limit: 789, # Integer | The number to limit by
  keys: ['inner_example'], # Array<String> | A list of keys to filter by
  workflows: ['inner_example'], # Array<String> | A list of workflow IDs to filter by
  statuses: [HatchetSdkRest::WorkflowRunStatus::PENDING], # Array<WorkflowRunStatus> | A list of workflow run statuses to filter by
  search: 'search_example', # String | The search query to filter for
  order_by_field: HatchetSdkRest::EventOrderByField::CREATED_AT, # EventOrderByField | What to order by
  order_by_direction: HatchetSdkRest::EventOrderByDirection::ASC, # EventOrderByDirection | The order direction
  additional_metadata: ['inner_example'], # Array<String> | A list of metadata key value pairs to filter by
  event_ids: ['inner_example'] # Array<String> | A list of event ids to filter by
}

begin
  # List events
  result = api_instance.event_list(tenant, opts)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling EventApi->event_list: #{e}"
end
```

#### Using the event_list_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<EventList>, Integer, Hash)> event_list_with_http_info(tenant, opts)

```ruby
begin
  # List events
  data, status_code, headers = api_instance.event_list_with_http_info(tenant, opts)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <EventList>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling EventApi->event_list_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **offset** | **Integer** | The number to skip | [optional] |
| **limit** | **Integer** | The number to limit by | [optional] |
| **keys** | [**Array&lt;String&gt;**](String.md) | A list of keys to filter by | [optional] |
| **workflows** | [**Array&lt;String&gt;**](String.md) | A list of workflow IDs to filter by | [optional] |
| **statuses** | [**Array&lt;WorkflowRunStatus&gt;**](WorkflowRunStatus.md) | A list of workflow run statuses to filter by | [optional] |
| **search** | **String** | The search query to filter for | [optional] |
| **order_by_field** | [**EventOrderByField**](.md) | What to order by | [optional] |
| **order_by_direction** | [**EventOrderByDirection**](.md) | The order direction | [optional] |
| **additional_metadata** | [**Array&lt;String&gt;**](String.md) | A list of metadata key value pairs to filter by | [optional] |
| **event_ids** | [**Array&lt;String&gt;**](String.md) | A list of event ids to filter by | [optional] |

### Return type

[**EventList**](EventList.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## event_update_cancel

> <EventUpdateCancel200Response> event_update_cancel(tenant, cancel_event_request)

Replay events

Cancels all runs for a list of events.

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'
# setup authorization
HatchetSdkRest.configure do |config|
  # Configure API key authorization: cookieAuth
  config.api_key['hatchet'] = 'YOUR API KEY'
  # Uncomment the following line to set a prefix for the API key, e.g. 'Bearer' (defaults to nil)
  # config.api_key_prefix['hatchet'] = 'Bearer'

  # Configure Bearer authorization: bearerAuth
  config.access_token = 'YOUR_BEARER_TOKEN'
end

api_instance = HatchetSdkRest::EventApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
cancel_event_request = HatchetSdkRest::CancelEventRequest.new({event_ids: ['bb214807-246e-43a5-a25d-41761d1cff9e']}) # CancelEventRequest | The event ids to replay

begin
  # Replay events
  result = api_instance.event_update_cancel(tenant, cancel_event_request)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling EventApi->event_update_cancel: #{e}"
end
```

#### Using the event_update_cancel_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<EventUpdateCancel200Response>, Integer, Hash)> event_update_cancel_with_http_info(tenant, cancel_event_request)

```ruby
begin
  # Replay events
  data, status_code, headers = api_instance.event_update_cancel_with_http_info(tenant, cancel_event_request)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <EventUpdateCancel200Response>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling EventApi->event_update_cancel_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **cancel_event_request** | [**CancelEventRequest**](CancelEventRequest.md) | The event ids to replay |  |

### Return type

[**EventUpdateCancel200Response**](EventUpdateCancel200Response.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json


## event_update_replay

> <EventList> event_update_replay(tenant, replay_event_request)

Replay events

Replays a list of events.

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'
# setup authorization
HatchetSdkRest.configure do |config|
  # Configure API key authorization: cookieAuth
  config.api_key['hatchet'] = 'YOUR API KEY'
  # Uncomment the following line to set a prefix for the API key, e.g. 'Bearer' (defaults to nil)
  # config.api_key_prefix['hatchet'] = 'Bearer'

  # Configure Bearer authorization: bearerAuth
  config.access_token = 'YOUR_BEARER_TOKEN'
end

api_instance = HatchetSdkRest::EventApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
replay_event_request = HatchetSdkRest::ReplayEventRequest.new({event_ids: ['bb214807-246e-43a5-a25d-41761d1cff9e']}) # ReplayEventRequest | The event ids to replay

begin
  # Replay events
  result = api_instance.event_update_replay(tenant, replay_event_request)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling EventApi->event_update_replay: #{e}"
end
```

#### Using the event_update_replay_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<EventList>, Integer, Hash)> event_update_replay_with_http_info(tenant, replay_event_request)

```ruby
begin
  # Replay events
  data, status_code, headers = api_instance.event_update_replay_with_http_info(tenant, replay_event_request)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <EventList>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling EventApi->event_update_replay_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **replay_event_request** | [**ReplayEventRequest**](ReplayEventRequest.md) | The event ids to replay |  |

### Return type

[**EventList**](EventList.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json


## v1_event_key_list

> <EventKeyList> v1_event_key_list(tenant)

List event keys

Lists all event keys for a tenant.

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'
# setup authorization
HatchetSdkRest.configure do |config|
  # Configure API key authorization: cookieAuth
  config.api_key['hatchet'] = 'YOUR API KEY'
  # Uncomment the following line to set a prefix for the API key, e.g. 'Bearer' (defaults to nil)
  # config.api_key_prefix['hatchet'] = 'Bearer'

  # Configure Bearer authorization: bearerAuth
  config.access_token = 'YOUR_BEARER_TOKEN'
end

api_instance = HatchetSdkRest::EventApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id

begin
  # List event keys
  result = api_instance.v1_event_key_list(tenant)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling EventApi->v1_event_key_list: #{e}"
end
```

#### Using the v1_event_key_list_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<EventKeyList>, Integer, Hash)> v1_event_key_list_with_http_info(tenant)

```ruby
begin
  # List event keys
  data, status_code, headers = api_instance.v1_event_key_list_with_http_info(tenant)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <EventKeyList>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling EventApi->v1_event_key_list_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |

### Return type

[**EventKeyList**](EventKeyList.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## v1_event_list

> <V1EventList> v1_event_list(tenant, opts)

List events

Lists all events for a tenant.

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'
# setup authorization
HatchetSdkRest.configure do |config|
  # Configure API key authorization: cookieAuth
  config.api_key['hatchet'] = 'YOUR API KEY'
  # Uncomment the following line to set a prefix for the API key, e.g. 'Bearer' (defaults to nil)
  # config.api_key_prefix['hatchet'] = 'Bearer'

  # Configure Bearer authorization: bearerAuth
  config.access_token = 'YOUR_BEARER_TOKEN'
end

api_instance = HatchetSdkRest::EventApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id
opts = {
  offset: 789, # Integer | The number to skip
  limit: 789, # Integer | The number to limit by
  keys: ['inner_example'], # Array<String> | A list of keys to filter by
  since: Time.parse('2013-10-20T19:20:30+01:00'), # Time | Consider events that occurred after this time
  _until: Time.parse('2013-10-20T19:20:30+01:00'), # Time | Consider events that occurred before this time
  workflow_ids: ['inner_example'], # Array<String> | Filter to events that are associated with a specific workflow run
  workflow_run_statuses: [HatchetSdkRest::V1TaskStatus::QUEUED], # Array<V1TaskStatus> | Filter to events that are associated with workflow runs matching a certain status
  event_ids: ['inner_example'], # Array<String> | Filter to specific events by their ids
  additional_metadata: ['inner_example'], # Array<String> | Filter by additional metadata on the events
  scopes: ['inner_example'] # Array<String> | The scopes to filter by
}

begin
  # List events
  result = api_instance.v1_event_list(tenant, opts)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling EventApi->v1_event_list: #{e}"
end
```

#### Using the v1_event_list_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<V1EventList>, Integer, Hash)> v1_event_list_with_http_info(tenant, opts)

```ruby
begin
  # List events
  data, status_code, headers = api_instance.v1_event_list_with_http_info(tenant, opts)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <V1EventList>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling EventApi->v1_event_list_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |
| **offset** | **Integer** | The number to skip | [optional] |
| **limit** | **Integer** | The number to limit by | [optional] |
| **keys** | [**Array&lt;String&gt;**](String.md) | A list of keys to filter by | [optional] |
| **since** | **Time** | Consider events that occurred after this time | [optional] |
| **_until** | **Time** | Consider events that occurred before this time | [optional] |
| **workflow_ids** | [**Array&lt;String&gt;**](String.md) | Filter to events that are associated with a specific workflow run | [optional] |
| **workflow_run_statuses** | [**Array&lt;V1TaskStatus&gt;**](V1TaskStatus.md) | Filter to events that are associated with workflow runs matching a certain status | [optional] |
| **event_ids** | [**Array&lt;String&gt;**](String.md) | Filter to specific events by their ids | [optional] |
| **additional_metadata** | [**Array&lt;String&gt;**](String.md) | Filter by additional metadata on the events | [optional] |
| **scopes** | [**Array&lt;String&gt;**](String.md) | The scopes to filter by | [optional] |

### Return type

[**V1EventList**](V1EventList.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

