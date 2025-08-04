# HatchetSdkRest::UserApi

All URIs are relative to *http://localhost*

| Method | HTTP request | Description |
| ------ | ------------ | ----------- |
| [**tenant_memberships_list**](UserApi.md#tenant_memberships_list) | **GET** /api/v1/users/memberships | List tenant memberships |
| [**user_create**](UserApi.md#user_create) | **POST** /api/v1/users/register | Register user |
| [**user_get_current**](UserApi.md#user_get_current) | **GET** /api/v1/users/current | Get current user |
| [**user_update_github_oauth_callback**](UserApi.md#user_update_github_oauth_callback) | **GET** /api/v1/users/github/callback | Complete OAuth flow |
| [**user_update_github_oauth_start**](UserApi.md#user_update_github_oauth_start) | **GET** /api/v1/users/github/start | Start OAuth flow |
| [**user_update_google_oauth_callback**](UserApi.md#user_update_google_oauth_callback) | **GET** /api/v1/users/google/callback | Complete OAuth flow |
| [**user_update_google_oauth_start**](UserApi.md#user_update_google_oauth_start) | **GET** /api/v1/users/google/start | Start OAuth flow |
| [**user_update_login**](UserApi.md#user_update_login) | **POST** /api/v1/users/login | Login user |
| [**user_update_logout**](UserApi.md#user_update_logout) | **POST** /api/v1/users/logout | Logout user |
| [**user_update_password**](UserApi.md#user_update_password) | **POST** /api/v1/users/password | Change user password |
| [**user_update_slack_oauth_callback**](UserApi.md#user_update_slack_oauth_callback) | **GET** /api/v1/users/slack/callback | Complete OAuth flow |
| [**user_update_slack_oauth_start**](UserApi.md#user_update_slack_oauth_start) | **GET** /api/v1/tenants/{tenant}/slack/start | Start OAuth flow |


## tenant_memberships_list

> <UserTenantMembershipsList> tenant_memberships_list

List tenant memberships

Lists all tenant memberships for the current user

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
end

api_instance = HatchetSdkRest::UserApi.new

begin
  # List tenant memberships
  result = api_instance.tenant_memberships_list
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling UserApi->tenant_memberships_list: #{e}"
end
```

#### Using the tenant_memberships_list_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<UserTenantMembershipsList>, Integer, Hash)> tenant_memberships_list_with_http_info

```ruby
begin
  # List tenant memberships
  data, status_code, headers = api_instance.tenant_memberships_list_with_http_info
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <UserTenantMembershipsList>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling UserApi->tenant_memberships_list_with_http_info: #{e}"
end
```

### Parameters

This endpoint does not need any parameter.

### Return type

[**UserTenantMembershipsList**](UserTenantMembershipsList.md)

### Authorization

[cookieAuth](../README.md#cookieAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## user_create

> <User> user_create(opts)

Register user

Registers a user.

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'

api_instance = HatchetSdkRest::UserApi.new
opts = {
  user_register_request: HatchetSdkRest::UserRegisterRequest.new({name: 'name_example', email: 'email_example', password: 'password_example'}) # UserRegisterRequest | 
}

begin
  # Register user
  result = api_instance.user_create(opts)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling UserApi->user_create: #{e}"
end
```

#### Using the user_create_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<User>, Integer, Hash)> user_create_with_http_info(opts)

```ruby
begin
  # Register user
  data, status_code, headers = api_instance.user_create_with_http_info(opts)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <User>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling UserApi->user_create_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **user_register_request** | [**UserRegisterRequest**](UserRegisterRequest.md) |  | [optional] |

### Return type

[**User**](User.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json


## user_get_current

> <User> user_get_current

Get current user

Gets the current user

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
end

api_instance = HatchetSdkRest::UserApi.new

begin
  # Get current user
  result = api_instance.user_get_current
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling UserApi->user_get_current: #{e}"
end
```

#### Using the user_get_current_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<User>, Integer, Hash)> user_get_current_with_http_info

```ruby
begin
  # Get current user
  data, status_code, headers = api_instance.user_get_current_with_http_info
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <User>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling UserApi->user_get_current_with_http_info: #{e}"
end
```

### Parameters

This endpoint does not need any parameter.

### Return type

[**User**](User.md)

### Authorization

[cookieAuth](../README.md#cookieAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## user_update_github_oauth_callback

> user_update_github_oauth_callback

Complete OAuth flow

Completes the OAuth flow

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'

api_instance = HatchetSdkRest::UserApi.new

begin
  # Complete OAuth flow
  api_instance.user_update_github_oauth_callback
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling UserApi->user_update_github_oauth_callback: #{e}"
end
```

#### Using the user_update_github_oauth_callback_with_http_info variant

This returns an Array which contains the response data (`nil` in this case), status code and headers.

> <Array(nil, Integer, Hash)> user_update_github_oauth_callback_with_http_info

```ruby
begin
  # Complete OAuth flow
  data, status_code, headers = api_instance.user_update_github_oauth_callback_with_http_info
  p status_code # => 2xx
  p headers # => { ... }
  p data # => nil
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling UserApi->user_update_github_oauth_callback_with_http_info: #{e}"
end
```

### Parameters

This endpoint does not need any parameter.

### Return type

nil (empty response body)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: Not defined


## user_update_github_oauth_start

> user_update_github_oauth_start

Start OAuth flow

Starts the OAuth flow

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'

api_instance = HatchetSdkRest::UserApi.new

begin
  # Start OAuth flow
  api_instance.user_update_github_oauth_start
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling UserApi->user_update_github_oauth_start: #{e}"
end
```

#### Using the user_update_github_oauth_start_with_http_info variant

This returns an Array which contains the response data (`nil` in this case), status code and headers.

> <Array(nil, Integer, Hash)> user_update_github_oauth_start_with_http_info

```ruby
begin
  # Start OAuth flow
  data, status_code, headers = api_instance.user_update_github_oauth_start_with_http_info
  p status_code # => 2xx
  p headers # => { ... }
  p data # => nil
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling UserApi->user_update_github_oauth_start_with_http_info: #{e}"
end
```

### Parameters

This endpoint does not need any parameter.

### Return type

nil (empty response body)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: Not defined


## user_update_google_oauth_callback

> user_update_google_oauth_callback

Complete OAuth flow

Completes the OAuth flow

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'

api_instance = HatchetSdkRest::UserApi.new

begin
  # Complete OAuth flow
  api_instance.user_update_google_oauth_callback
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling UserApi->user_update_google_oauth_callback: #{e}"
end
```

#### Using the user_update_google_oauth_callback_with_http_info variant

This returns an Array which contains the response data (`nil` in this case), status code and headers.

> <Array(nil, Integer, Hash)> user_update_google_oauth_callback_with_http_info

```ruby
begin
  # Complete OAuth flow
  data, status_code, headers = api_instance.user_update_google_oauth_callback_with_http_info
  p status_code # => 2xx
  p headers # => { ... }
  p data # => nil
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling UserApi->user_update_google_oauth_callback_with_http_info: #{e}"
end
```

### Parameters

This endpoint does not need any parameter.

### Return type

nil (empty response body)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: Not defined


## user_update_google_oauth_start

> user_update_google_oauth_start

Start OAuth flow

Starts the OAuth flow

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'

api_instance = HatchetSdkRest::UserApi.new

begin
  # Start OAuth flow
  api_instance.user_update_google_oauth_start
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling UserApi->user_update_google_oauth_start: #{e}"
end
```

#### Using the user_update_google_oauth_start_with_http_info variant

This returns an Array which contains the response data (`nil` in this case), status code and headers.

> <Array(nil, Integer, Hash)> user_update_google_oauth_start_with_http_info

```ruby
begin
  # Start OAuth flow
  data, status_code, headers = api_instance.user_update_google_oauth_start_with_http_info
  p status_code # => 2xx
  p headers # => { ... }
  p data # => nil
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling UserApi->user_update_google_oauth_start_with_http_info: #{e}"
end
```

### Parameters

This endpoint does not need any parameter.

### Return type

nil (empty response body)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: Not defined


## user_update_login

> <User> user_update_login(opts)

Login user

Logs in a user.

### Examples

```ruby
require 'time'
require 'hatchet-sdk-rest'

api_instance = HatchetSdkRest::UserApi.new
opts = {
  user_login_request: HatchetSdkRest::UserLoginRequest.new({email: 'email_example', password: 'password_example'}) # UserLoginRequest | 
}

begin
  # Login user
  result = api_instance.user_update_login(opts)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling UserApi->user_update_login: #{e}"
end
```

#### Using the user_update_login_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<User>, Integer, Hash)> user_update_login_with_http_info(opts)

```ruby
begin
  # Login user
  data, status_code, headers = api_instance.user_update_login_with_http_info(opts)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <User>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling UserApi->user_update_login_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **user_login_request** | [**UserLoginRequest**](UserLoginRequest.md) |  | [optional] |

### Return type

[**User**](User.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json


## user_update_logout

> <User> user_update_logout

Logout user

Logs out a user.

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
end

api_instance = HatchetSdkRest::UserApi.new

begin
  # Logout user
  result = api_instance.user_update_logout
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling UserApi->user_update_logout: #{e}"
end
```

#### Using the user_update_logout_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<User>, Integer, Hash)> user_update_logout_with_http_info

```ruby
begin
  # Logout user
  data, status_code, headers = api_instance.user_update_logout_with_http_info
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <User>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling UserApi->user_update_logout_with_http_info: #{e}"
end
```

### Parameters

This endpoint does not need any parameter.

### Return type

[**User**](User.md)

### Authorization

[cookieAuth](../README.md#cookieAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json


## user_update_password

> <User> user_update_password(opts)

Change user password

Update a user password.

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
end

api_instance = HatchetSdkRest::UserApi.new
opts = {
  user_change_password_request: HatchetSdkRest::UserChangePasswordRequest.new({password: 'password_example', new_password: 'new_password_example'}) # UserChangePasswordRequest | 
}

begin
  # Change user password
  result = api_instance.user_update_password(opts)
  p result
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling UserApi->user_update_password: #{e}"
end
```

#### Using the user_update_password_with_http_info variant

This returns an Array which contains the response data, status code and headers.

> <Array(<User>, Integer, Hash)> user_update_password_with_http_info(opts)

```ruby
begin
  # Change user password
  data, status_code, headers = api_instance.user_update_password_with_http_info(opts)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => <User>
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling UserApi->user_update_password_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **user_change_password_request** | [**UserChangePasswordRequest**](UserChangePasswordRequest.md) |  | [optional] |

### Return type

[**User**](User.md)

### Authorization

[cookieAuth](../README.md#cookieAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json


## user_update_slack_oauth_callback

> user_update_slack_oauth_callback

Complete OAuth flow

Completes the OAuth flow

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
end

api_instance = HatchetSdkRest::UserApi.new

begin
  # Complete OAuth flow
  api_instance.user_update_slack_oauth_callback
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling UserApi->user_update_slack_oauth_callback: #{e}"
end
```

#### Using the user_update_slack_oauth_callback_with_http_info variant

This returns an Array which contains the response data (`nil` in this case), status code and headers.

> <Array(nil, Integer, Hash)> user_update_slack_oauth_callback_with_http_info

```ruby
begin
  # Complete OAuth flow
  data, status_code, headers = api_instance.user_update_slack_oauth_callback_with_http_info
  p status_code # => 2xx
  p headers # => { ... }
  p data # => nil
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling UserApi->user_update_slack_oauth_callback_with_http_info: #{e}"
end
```

### Parameters

This endpoint does not need any parameter.

### Return type

nil (empty response body)

### Authorization

[cookieAuth](../README.md#cookieAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: Not defined


## user_update_slack_oauth_start

> user_update_slack_oauth_start(tenant)

Start OAuth flow

Starts the OAuth flow

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
end

api_instance = HatchetSdkRest::UserApi.new
tenant = '38400000-8cf0-11bd-b23e-10b96e4ef00d' # String | The tenant id

begin
  # Start OAuth flow
  api_instance.user_update_slack_oauth_start(tenant)
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling UserApi->user_update_slack_oauth_start: #{e}"
end
```

#### Using the user_update_slack_oauth_start_with_http_info variant

This returns an Array which contains the response data (`nil` in this case), status code and headers.

> <Array(nil, Integer, Hash)> user_update_slack_oauth_start_with_http_info(tenant)

```ruby
begin
  # Start OAuth flow
  data, status_code, headers = api_instance.user_update_slack_oauth_start_with_http_info(tenant)
  p status_code # => 2xx
  p headers # => { ... }
  p data # => nil
rescue HatchetSdkRest::ApiError => e
  puts "Error when calling UserApi->user_update_slack_oauth_start_with_http_info: #{e}"
end
```

### Parameters

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **tenant** | **String** | The tenant id |  |

### Return type

nil (empty response body)

### Authorization

[cookieAuth](../README.md#cookieAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: Not defined

