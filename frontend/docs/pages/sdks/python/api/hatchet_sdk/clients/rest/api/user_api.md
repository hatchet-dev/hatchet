# Table of Contents

* [hatchet\_sdk.clients.rest.api.user\_api](#hatchet_sdk.clients.rest.api.user_api)
  * [UserApi](#hatchet_sdk.clients.rest.api.user_api.UserApi)
    * [tenant\_memberships\_list](#hatchet_sdk.clients.rest.api.user_api.UserApi.tenant_memberships_list)
    * [tenant\_memberships\_list\_with\_http\_info](#hatchet_sdk.clients.rest.api.user_api.UserApi.tenant_memberships_list_with_http_info)
    * [tenant\_memberships\_list\_without\_preload\_content](#hatchet_sdk.clients.rest.api.user_api.UserApi.tenant_memberships_list_without_preload_content)
    * [user\_create](#hatchet_sdk.clients.rest.api.user_api.UserApi.user_create)
    * [user\_create\_with\_http\_info](#hatchet_sdk.clients.rest.api.user_api.UserApi.user_create_with_http_info)
    * [user\_create\_without\_preload\_content](#hatchet_sdk.clients.rest.api.user_api.UserApi.user_create_without_preload_content)
    * [user\_get\_current](#hatchet_sdk.clients.rest.api.user_api.UserApi.user_get_current)
    * [user\_get\_current\_with\_http\_info](#hatchet_sdk.clients.rest.api.user_api.UserApi.user_get_current_with_http_info)
    * [user\_get\_current\_without\_preload\_content](#hatchet_sdk.clients.rest.api.user_api.UserApi.user_get_current_without_preload_content)
    * [user\_update\_github\_oauth\_callback](#hatchet_sdk.clients.rest.api.user_api.UserApi.user_update_github_oauth_callback)
    * [user\_update\_github\_oauth\_callback\_with\_http\_info](#hatchet_sdk.clients.rest.api.user_api.UserApi.user_update_github_oauth_callback_with_http_info)
    * [user\_update\_github\_oauth\_callback\_without\_preload\_content](#hatchet_sdk.clients.rest.api.user_api.UserApi.user_update_github_oauth_callback_without_preload_content)
    * [user\_update\_github\_oauth\_start](#hatchet_sdk.clients.rest.api.user_api.UserApi.user_update_github_oauth_start)
    * [user\_update\_github\_oauth\_start\_with\_http\_info](#hatchet_sdk.clients.rest.api.user_api.UserApi.user_update_github_oauth_start_with_http_info)
    * [user\_update\_github\_oauth\_start\_without\_preload\_content](#hatchet_sdk.clients.rest.api.user_api.UserApi.user_update_github_oauth_start_without_preload_content)
    * [user\_update\_google\_oauth\_callback](#hatchet_sdk.clients.rest.api.user_api.UserApi.user_update_google_oauth_callback)
    * [user\_update\_google\_oauth\_callback\_with\_http\_info](#hatchet_sdk.clients.rest.api.user_api.UserApi.user_update_google_oauth_callback_with_http_info)
    * [user\_update\_google\_oauth\_callback\_without\_preload\_content](#hatchet_sdk.clients.rest.api.user_api.UserApi.user_update_google_oauth_callback_without_preload_content)
    * [user\_update\_google\_oauth\_start](#hatchet_sdk.clients.rest.api.user_api.UserApi.user_update_google_oauth_start)
    * [user\_update\_google\_oauth\_start\_with\_http\_info](#hatchet_sdk.clients.rest.api.user_api.UserApi.user_update_google_oauth_start_with_http_info)
    * [user\_update\_google\_oauth\_start\_without\_preload\_content](#hatchet_sdk.clients.rest.api.user_api.UserApi.user_update_google_oauth_start_without_preload_content)
    * [user\_update\_login](#hatchet_sdk.clients.rest.api.user_api.UserApi.user_update_login)
    * [user\_update\_login\_with\_http\_info](#hatchet_sdk.clients.rest.api.user_api.UserApi.user_update_login_with_http_info)
    * [user\_update\_login\_without\_preload\_content](#hatchet_sdk.clients.rest.api.user_api.UserApi.user_update_login_without_preload_content)
    * [user\_update\_logout](#hatchet_sdk.clients.rest.api.user_api.UserApi.user_update_logout)
    * [user\_update\_logout\_with\_http\_info](#hatchet_sdk.clients.rest.api.user_api.UserApi.user_update_logout_with_http_info)
    * [user\_update\_logout\_without\_preload\_content](#hatchet_sdk.clients.rest.api.user_api.UserApi.user_update_logout_without_preload_content)
    * [user\_update\_password](#hatchet_sdk.clients.rest.api.user_api.UserApi.user_update_password)
    * [user\_update\_password\_with\_http\_info](#hatchet_sdk.clients.rest.api.user_api.UserApi.user_update_password_with_http_info)
    * [user\_update\_password\_without\_preload\_content](#hatchet_sdk.clients.rest.api.user_api.UserApi.user_update_password_without_preload_content)
    * [user\_update\_slack\_oauth\_callback](#hatchet_sdk.clients.rest.api.user_api.UserApi.user_update_slack_oauth_callback)
    * [user\_update\_slack\_oauth\_callback\_with\_http\_info](#hatchet_sdk.clients.rest.api.user_api.UserApi.user_update_slack_oauth_callback_with_http_info)
    * [user\_update\_slack\_oauth\_callback\_without\_preload\_content](#hatchet_sdk.clients.rest.api.user_api.UserApi.user_update_slack_oauth_callback_without_preload_content)
    * [user\_update\_slack\_oauth\_start](#hatchet_sdk.clients.rest.api.user_api.UserApi.user_update_slack_oauth_start)
    * [user\_update\_slack\_oauth\_start\_with\_http\_info](#hatchet_sdk.clients.rest.api.user_api.UserApi.user_update_slack_oauth_start_with_http_info)
    * [user\_update\_slack\_oauth\_start\_without\_preload\_content](#hatchet_sdk.clients.rest.api.user_api.UserApi.user_update_slack_oauth_start_without_preload_content)

---
sidebar_label: user_api
title: hatchet_sdk.clients.rest.api.user_api
---

Hatchet API

The Hatchet API

The version of the OpenAPI document: 1.0.0
Generated by OpenAPI Generator (https://openapi-generator.tech)

Do not edit the class manually.

## UserApi

NOTE: This class is auto generated by OpenAPI Generator
Ref: https://openapi-generator.tech

Do not edit the class manually.

#### tenant\_memberships\_list

```python
@validate_call
def tenant_memberships_list(
    _request_timeout: Union[
        None,
        Annotated[StrictFloat, Field(gt=0)],
        Tuple[Annotated[StrictFloat, Field(gt=0)], Annotated[StrictFloat,
                                                             Field(gt=0)]],
    ] = None,
    _request_auth: Optional[Dict[StrictStr, Any]] = None,
    _content_type: Optional[StrictStr] = None,
    _headers: Optional[Dict[StrictStr, Any]] = None,
    _host_index: Annotated[StrictInt, Field(ge=0, le=0)] = 0
) -> UserTenantMembershipsList
```

List tenant memberships

Lists all tenant memberships for the current user

**Arguments**:

- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `_headers` (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `_host_index` (`int, optional`): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### tenant\_memberships\_list\_with\_http\_info

```python
@validate_call
def tenant_memberships_list_with_http_info(
    _request_timeout: Union[
        None,
        Annotated[StrictFloat, Field(gt=0)],
        Tuple[Annotated[StrictFloat, Field(gt=0)], Annotated[StrictFloat,
                                                             Field(gt=0)]],
    ] = None,
    _request_auth: Optional[Dict[StrictStr, Any]] = None,
    _content_type: Optional[StrictStr] = None,
    _headers: Optional[Dict[StrictStr, Any]] = None,
    _host_index: Annotated[StrictInt, Field(ge=0, le=0)] = 0
) -> ApiResponse[UserTenantMembershipsList]
```

List tenant memberships

Lists all tenant memberships for the current user

**Arguments**:

- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `_headers` (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `_host_index` (`int, optional`): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### tenant\_memberships\_list\_without\_preload\_content

```python
@validate_call
def tenant_memberships_list_without_preload_content(
    _request_timeout: Union[
        None,
        Annotated[StrictFloat, Field(gt=0)],
        Tuple[Annotated[StrictFloat, Field(gt=0)], Annotated[StrictFloat,
                                                             Field(gt=0)]],
    ] = None,
    _request_auth: Optional[Dict[StrictStr, Any]] = None,
    _content_type: Optional[StrictStr] = None,
    _headers: Optional[Dict[StrictStr, Any]] = None,
    _host_index: Annotated[StrictInt,
                           Field(ge=0, le=0)] = 0) -> RESTResponseType
```

List tenant memberships

Lists all tenant memberships for the current user

**Arguments**:

- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `_headers` (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `_host_index` (`int, optional`): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### user\_create

```python
@validate_call
def user_create(
        user_register_request: Optional[UserRegisterRequest] = None,
        _request_timeout: Union[
            None,
            Annotated[StrictFloat, Field(gt=0)],
            Tuple[Annotated[StrictFloat, Field(gt=0)], Annotated[StrictFloat,
                                                                 Field(gt=0)]],
        ] = None,
        _request_auth: Optional[Dict[StrictStr, Any]] = None,
        _content_type: Optional[StrictStr] = None,
        _headers: Optional[Dict[StrictStr, Any]] = None,
        _host_index: Annotated[StrictInt, Field(ge=0, le=0)] = 0) -> User
```

Register user

Registers a user.

**Arguments**:

- `user_register_request` (`UserRegisterRequest`): 
- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `_headers` (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `UserRegisterRequest`0 (`UserRegisterRequest`1): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### user\_create\_with\_http\_info

```python
@validate_call
def user_create_with_http_info(
    user_register_request: Optional[UserRegisterRequest] = None,
    _request_timeout: Union[
        None,
        Annotated[StrictFloat, Field(gt=0)],
        Tuple[Annotated[StrictFloat, Field(gt=0)], Annotated[StrictFloat,
                                                             Field(gt=0)]],
    ] = None,
    _request_auth: Optional[Dict[StrictStr, Any]] = None,
    _content_type: Optional[StrictStr] = None,
    _headers: Optional[Dict[StrictStr, Any]] = None,
    _host_index: Annotated[StrictInt,
                           Field(ge=0, le=0)] = 0) -> ApiResponse[User]
```

Register user

Registers a user.

**Arguments**:

- `user_register_request` (`UserRegisterRequest`): 
- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `_headers` (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `UserRegisterRequest`0 (`UserRegisterRequest`1): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### user\_create\_without\_preload\_content

```python
@validate_call
def user_create_without_preload_content(
    user_register_request: Optional[UserRegisterRequest] = None,
    _request_timeout: Union[
        None,
        Annotated[StrictFloat, Field(gt=0)],
        Tuple[Annotated[StrictFloat, Field(gt=0)], Annotated[StrictFloat,
                                                             Field(gt=0)]],
    ] = None,
    _request_auth: Optional[Dict[StrictStr, Any]] = None,
    _content_type: Optional[StrictStr] = None,
    _headers: Optional[Dict[StrictStr, Any]] = None,
    _host_index: Annotated[StrictInt,
                           Field(ge=0, le=0)] = 0) -> RESTResponseType
```

Register user

Registers a user.

**Arguments**:

- `user_register_request` (`UserRegisterRequest`): 
- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `_headers` (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `UserRegisterRequest`0 (`UserRegisterRequest`1): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### user\_get\_current

```python
@validate_call
def user_get_current(
        _request_timeout: Union[
            None,
            Annotated[StrictFloat, Field(gt=0)],
            Tuple[Annotated[StrictFloat, Field(gt=0)], Annotated[StrictFloat,
                                                                 Field(gt=0)]],
        ] = None,
        _request_auth: Optional[Dict[StrictStr, Any]] = None,
        _content_type: Optional[StrictStr] = None,
        _headers: Optional[Dict[StrictStr, Any]] = None,
        _host_index: Annotated[StrictInt, Field(ge=0, le=0)] = 0) -> User
```

Get current user

Gets the current user

**Arguments**:

- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `_headers` (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `_host_index` (`int, optional`): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### user\_get\_current\_with\_http\_info

```python
@validate_call
def user_get_current_with_http_info(
    _request_timeout: Union[
        None,
        Annotated[StrictFloat, Field(gt=0)],
        Tuple[Annotated[StrictFloat, Field(gt=0)], Annotated[StrictFloat,
                                                             Field(gt=0)]],
    ] = None,
    _request_auth: Optional[Dict[StrictStr, Any]] = None,
    _content_type: Optional[StrictStr] = None,
    _headers: Optional[Dict[StrictStr, Any]] = None,
    _host_index: Annotated[StrictInt,
                           Field(ge=0, le=0)] = 0) -> ApiResponse[User]
```

Get current user

Gets the current user

**Arguments**:

- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `_headers` (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `_host_index` (`int, optional`): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### user\_get\_current\_without\_preload\_content

```python
@validate_call
def user_get_current_without_preload_content(
    _request_timeout: Union[
        None,
        Annotated[StrictFloat, Field(gt=0)],
        Tuple[Annotated[StrictFloat, Field(gt=0)], Annotated[StrictFloat,
                                                             Field(gt=0)]],
    ] = None,
    _request_auth: Optional[Dict[StrictStr, Any]] = None,
    _content_type: Optional[StrictStr] = None,
    _headers: Optional[Dict[StrictStr, Any]] = None,
    _host_index: Annotated[StrictInt,
                           Field(ge=0, le=0)] = 0) -> RESTResponseType
```

Get current user

Gets the current user

**Arguments**:

- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `_headers` (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `_host_index` (`int, optional`): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### user\_update\_github\_oauth\_callback

```python
@validate_call
def user_update_github_oauth_callback(
        _request_timeout: Union[
            None,
            Annotated[StrictFloat, Field(gt=0)],
            Tuple[Annotated[StrictFloat, Field(gt=0)], Annotated[StrictFloat,
                                                                 Field(gt=0)]],
        ] = None,
        _request_auth: Optional[Dict[StrictStr, Any]] = None,
        _content_type: Optional[StrictStr] = None,
        _headers: Optional[Dict[StrictStr, Any]] = None,
        _host_index: Annotated[StrictInt, Field(ge=0, le=0)] = 0) -> None
```

Complete OAuth flow

Completes the OAuth flow

**Arguments**:

- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `_headers` (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `_host_index` (`int, optional`): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### user\_update\_github\_oauth\_callback\_with\_http\_info

```python
@validate_call
def user_update_github_oauth_callback_with_http_info(
    _request_timeout: Union[
        None,
        Annotated[StrictFloat, Field(gt=0)],
        Tuple[Annotated[StrictFloat, Field(gt=0)], Annotated[StrictFloat,
                                                             Field(gt=0)]],
    ] = None,
    _request_auth: Optional[Dict[StrictStr, Any]] = None,
    _content_type: Optional[StrictStr] = None,
    _headers: Optional[Dict[StrictStr, Any]] = None,
    _host_index: Annotated[StrictInt,
                           Field(ge=0, le=0)] = 0) -> ApiResponse[None]
```

Complete OAuth flow

Completes the OAuth flow

**Arguments**:

- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `_headers` (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `_host_index` (`int, optional`): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### user\_update\_github\_oauth\_callback\_without\_preload\_content

```python
@validate_call
def user_update_github_oauth_callback_without_preload_content(
    _request_timeout: Union[
        None,
        Annotated[StrictFloat, Field(gt=0)],
        Tuple[Annotated[StrictFloat, Field(gt=0)], Annotated[StrictFloat,
                                                             Field(gt=0)]],
    ] = None,
    _request_auth: Optional[Dict[StrictStr, Any]] = None,
    _content_type: Optional[StrictStr] = None,
    _headers: Optional[Dict[StrictStr, Any]] = None,
    _host_index: Annotated[StrictInt,
                           Field(ge=0, le=0)] = 0) -> RESTResponseType
```

Complete OAuth flow

Completes the OAuth flow

**Arguments**:

- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `_headers` (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `_host_index` (`int, optional`): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### user\_update\_github\_oauth\_start

```python
@validate_call
def user_update_github_oauth_start(
        _request_timeout: Union[
            None,
            Annotated[StrictFloat, Field(gt=0)],
            Tuple[Annotated[StrictFloat, Field(gt=0)], Annotated[StrictFloat,
                                                                 Field(gt=0)]],
        ] = None,
        _request_auth: Optional[Dict[StrictStr, Any]] = None,
        _content_type: Optional[StrictStr] = None,
        _headers: Optional[Dict[StrictStr, Any]] = None,
        _host_index: Annotated[StrictInt, Field(ge=0, le=0)] = 0) -> None
```

Start OAuth flow

Starts the OAuth flow

**Arguments**:

- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `_headers` (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `_host_index` (`int, optional`): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### user\_update\_github\_oauth\_start\_with\_http\_info

```python
@validate_call
def user_update_github_oauth_start_with_http_info(
    _request_timeout: Union[
        None,
        Annotated[StrictFloat, Field(gt=0)],
        Tuple[Annotated[StrictFloat, Field(gt=0)], Annotated[StrictFloat,
                                                             Field(gt=0)]],
    ] = None,
    _request_auth: Optional[Dict[StrictStr, Any]] = None,
    _content_type: Optional[StrictStr] = None,
    _headers: Optional[Dict[StrictStr, Any]] = None,
    _host_index: Annotated[StrictInt,
                           Field(ge=0, le=0)] = 0) -> ApiResponse[None]
```

Start OAuth flow

Starts the OAuth flow

**Arguments**:

- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `_headers` (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `_host_index` (`int, optional`): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### user\_update\_github\_oauth\_start\_without\_preload\_content

```python
@validate_call
def user_update_github_oauth_start_without_preload_content(
    _request_timeout: Union[
        None,
        Annotated[StrictFloat, Field(gt=0)],
        Tuple[Annotated[StrictFloat, Field(gt=0)], Annotated[StrictFloat,
                                                             Field(gt=0)]],
    ] = None,
    _request_auth: Optional[Dict[StrictStr, Any]] = None,
    _content_type: Optional[StrictStr] = None,
    _headers: Optional[Dict[StrictStr, Any]] = None,
    _host_index: Annotated[StrictInt,
                           Field(ge=0, le=0)] = 0) -> RESTResponseType
```

Start OAuth flow

Starts the OAuth flow

**Arguments**:

- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `_headers` (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `_host_index` (`int, optional`): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### user\_update\_google\_oauth\_callback

```python
@validate_call
def user_update_google_oauth_callback(
        _request_timeout: Union[
            None,
            Annotated[StrictFloat, Field(gt=0)],
            Tuple[Annotated[StrictFloat, Field(gt=0)], Annotated[StrictFloat,
                                                                 Field(gt=0)]],
        ] = None,
        _request_auth: Optional[Dict[StrictStr, Any]] = None,
        _content_type: Optional[StrictStr] = None,
        _headers: Optional[Dict[StrictStr, Any]] = None,
        _host_index: Annotated[StrictInt, Field(ge=0, le=0)] = 0) -> None
```

Complete OAuth flow

Completes the OAuth flow

**Arguments**:

- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `_headers` (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `_host_index` (`int, optional`): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### user\_update\_google\_oauth\_callback\_with\_http\_info

```python
@validate_call
def user_update_google_oauth_callback_with_http_info(
    _request_timeout: Union[
        None,
        Annotated[StrictFloat, Field(gt=0)],
        Tuple[Annotated[StrictFloat, Field(gt=0)], Annotated[StrictFloat,
                                                             Field(gt=0)]],
    ] = None,
    _request_auth: Optional[Dict[StrictStr, Any]] = None,
    _content_type: Optional[StrictStr] = None,
    _headers: Optional[Dict[StrictStr, Any]] = None,
    _host_index: Annotated[StrictInt,
                           Field(ge=0, le=0)] = 0) -> ApiResponse[None]
```

Complete OAuth flow

Completes the OAuth flow

**Arguments**:

- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `_headers` (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `_host_index` (`int, optional`): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### user\_update\_google\_oauth\_callback\_without\_preload\_content

```python
@validate_call
def user_update_google_oauth_callback_without_preload_content(
    _request_timeout: Union[
        None,
        Annotated[StrictFloat, Field(gt=0)],
        Tuple[Annotated[StrictFloat, Field(gt=0)], Annotated[StrictFloat,
                                                             Field(gt=0)]],
    ] = None,
    _request_auth: Optional[Dict[StrictStr, Any]] = None,
    _content_type: Optional[StrictStr] = None,
    _headers: Optional[Dict[StrictStr, Any]] = None,
    _host_index: Annotated[StrictInt,
                           Field(ge=0, le=0)] = 0) -> RESTResponseType
```

Complete OAuth flow

Completes the OAuth flow

**Arguments**:

- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `_headers` (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `_host_index` (`int, optional`): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### user\_update\_google\_oauth\_start

```python
@validate_call
def user_update_google_oauth_start(
        _request_timeout: Union[
            None,
            Annotated[StrictFloat, Field(gt=0)],
            Tuple[Annotated[StrictFloat, Field(gt=0)], Annotated[StrictFloat,
                                                                 Field(gt=0)]],
        ] = None,
        _request_auth: Optional[Dict[StrictStr, Any]] = None,
        _content_type: Optional[StrictStr] = None,
        _headers: Optional[Dict[StrictStr, Any]] = None,
        _host_index: Annotated[StrictInt, Field(ge=0, le=0)] = 0) -> None
```

Start OAuth flow

Starts the OAuth flow

**Arguments**:

- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `_headers` (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `_host_index` (`int, optional`): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### user\_update\_google\_oauth\_start\_with\_http\_info

```python
@validate_call
def user_update_google_oauth_start_with_http_info(
    _request_timeout: Union[
        None,
        Annotated[StrictFloat, Field(gt=0)],
        Tuple[Annotated[StrictFloat, Field(gt=0)], Annotated[StrictFloat,
                                                             Field(gt=0)]],
    ] = None,
    _request_auth: Optional[Dict[StrictStr, Any]] = None,
    _content_type: Optional[StrictStr] = None,
    _headers: Optional[Dict[StrictStr, Any]] = None,
    _host_index: Annotated[StrictInt,
                           Field(ge=0, le=0)] = 0) -> ApiResponse[None]
```

Start OAuth flow

Starts the OAuth flow

**Arguments**:

- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `_headers` (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `_host_index` (`int, optional`): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### user\_update\_google\_oauth\_start\_without\_preload\_content

```python
@validate_call
def user_update_google_oauth_start_without_preload_content(
    _request_timeout: Union[
        None,
        Annotated[StrictFloat, Field(gt=0)],
        Tuple[Annotated[StrictFloat, Field(gt=0)], Annotated[StrictFloat,
                                                             Field(gt=0)]],
    ] = None,
    _request_auth: Optional[Dict[StrictStr, Any]] = None,
    _content_type: Optional[StrictStr] = None,
    _headers: Optional[Dict[StrictStr, Any]] = None,
    _host_index: Annotated[StrictInt,
                           Field(ge=0, le=0)] = 0) -> RESTResponseType
```

Start OAuth flow

Starts the OAuth flow

**Arguments**:

- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `_headers` (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `_host_index` (`int, optional`): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### user\_update\_login

```python
@validate_call
def user_update_login(
        user_login_request: Optional[UserLoginRequest] = None,
        _request_timeout: Union[
            None,
            Annotated[StrictFloat, Field(gt=0)],
            Tuple[Annotated[StrictFloat, Field(gt=0)], Annotated[StrictFloat,
                                                                 Field(gt=0)]],
        ] = None,
        _request_auth: Optional[Dict[StrictStr, Any]] = None,
        _content_type: Optional[StrictStr] = None,
        _headers: Optional[Dict[StrictStr, Any]] = None,
        _host_index: Annotated[StrictInt, Field(ge=0, le=0)] = 0) -> User
```

Login user

Logs in a user.

**Arguments**:

- `user_login_request` (`UserLoginRequest`): 
- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `_headers` (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `UserLoginRequest`0 (`UserLoginRequest`1): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### user\_update\_login\_with\_http\_info

```python
@validate_call
def user_update_login_with_http_info(
    user_login_request: Optional[UserLoginRequest] = None,
    _request_timeout: Union[
        None,
        Annotated[StrictFloat, Field(gt=0)],
        Tuple[Annotated[StrictFloat, Field(gt=0)], Annotated[StrictFloat,
                                                             Field(gt=0)]],
    ] = None,
    _request_auth: Optional[Dict[StrictStr, Any]] = None,
    _content_type: Optional[StrictStr] = None,
    _headers: Optional[Dict[StrictStr, Any]] = None,
    _host_index: Annotated[StrictInt,
                           Field(ge=0, le=0)] = 0) -> ApiResponse[User]
```

Login user

Logs in a user.

**Arguments**:

- `user_login_request` (`UserLoginRequest`): 
- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `_headers` (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `UserLoginRequest`0 (`UserLoginRequest`1): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### user\_update\_login\_without\_preload\_content

```python
@validate_call
def user_update_login_without_preload_content(
    user_login_request: Optional[UserLoginRequest] = None,
    _request_timeout: Union[
        None,
        Annotated[StrictFloat, Field(gt=0)],
        Tuple[Annotated[StrictFloat, Field(gt=0)], Annotated[StrictFloat,
                                                             Field(gt=0)]],
    ] = None,
    _request_auth: Optional[Dict[StrictStr, Any]] = None,
    _content_type: Optional[StrictStr] = None,
    _headers: Optional[Dict[StrictStr, Any]] = None,
    _host_index: Annotated[StrictInt,
                           Field(ge=0, le=0)] = 0) -> RESTResponseType
```

Login user

Logs in a user.

**Arguments**:

- `user_login_request` (`UserLoginRequest`): 
- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `_headers` (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `UserLoginRequest`0 (`UserLoginRequest`1): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### user\_update\_logout

```python
@validate_call
def user_update_logout(
        _request_timeout: Union[
            None,
            Annotated[StrictFloat, Field(gt=0)],
            Tuple[Annotated[StrictFloat, Field(gt=0)], Annotated[StrictFloat,
                                                                 Field(gt=0)]],
        ] = None,
        _request_auth: Optional[Dict[StrictStr, Any]] = None,
        _content_type: Optional[StrictStr] = None,
        _headers: Optional[Dict[StrictStr, Any]] = None,
        _host_index: Annotated[StrictInt, Field(ge=0, le=0)] = 0) -> User
```

Logout user

Logs out a user.

**Arguments**:

- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `_headers` (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `_host_index` (`int, optional`): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### user\_update\_logout\_with\_http\_info

```python
@validate_call
def user_update_logout_with_http_info(
    _request_timeout: Union[
        None,
        Annotated[StrictFloat, Field(gt=0)],
        Tuple[Annotated[StrictFloat, Field(gt=0)], Annotated[StrictFloat,
                                                             Field(gt=0)]],
    ] = None,
    _request_auth: Optional[Dict[StrictStr, Any]] = None,
    _content_type: Optional[StrictStr] = None,
    _headers: Optional[Dict[StrictStr, Any]] = None,
    _host_index: Annotated[StrictInt,
                           Field(ge=0, le=0)] = 0) -> ApiResponse[User]
```

Logout user

Logs out a user.

**Arguments**:

- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `_headers` (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `_host_index` (`int, optional`): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### user\_update\_logout\_without\_preload\_content

```python
@validate_call
def user_update_logout_without_preload_content(
    _request_timeout: Union[
        None,
        Annotated[StrictFloat, Field(gt=0)],
        Tuple[Annotated[StrictFloat, Field(gt=0)], Annotated[StrictFloat,
                                                             Field(gt=0)]],
    ] = None,
    _request_auth: Optional[Dict[StrictStr, Any]] = None,
    _content_type: Optional[StrictStr] = None,
    _headers: Optional[Dict[StrictStr, Any]] = None,
    _host_index: Annotated[StrictInt,
                           Field(ge=0, le=0)] = 0) -> RESTResponseType
```

Logout user

Logs out a user.

**Arguments**:

- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `_headers` (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `_host_index` (`int, optional`): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### user\_update\_password

```python
@validate_call
def user_update_password(
        user_change_password_request: Optional[
            UserChangePasswordRequest] = None,
        _request_timeout: Union[
            None,
            Annotated[StrictFloat, Field(gt=0)],
            Tuple[Annotated[StrictFloat, Field(gt=0)], Annotated[StrictFloat,
                                                                 Field(gt=0)]],
        ] = None,
        _request_auth: Optional[Dict[StrictStr, Any]] = None,
        _content_type: Optional[StrictStr] = None,
        _headers: Optional[Dict[StrictStr, Any]] = None,
        _host_index: Annotated[StrictInt, Field(ge=0, le=0)] = 0) -> User
```

Change user password

Update a user password.

**Arguments**:

- `user_change_password_request` (`UserChangePasswordRequest`): 
- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `_headers` (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `UserChangePasswordRequest`0 (`UserChangePasswordRequest`1): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### user\_update\_password\_with\_http\_info

```python
@validate_call
def user_update_password_with_http_info(
    user_change_password_request: Optional[UserChangePasswordRequest] = None,
    _request_timeout: Union[
        None,
        Annotated[StrictFloat, Field(gt=0)],
        Tuple[Annotated[StrictFloat, Field(gt=0)], Annotated[StrictFloat,
                                                             Field(gt=0)]],
    ] = None,
    _request_auth: Optional[Dict[StrictStr, Any]] = None,
    _content_type: Optional[StrictStr] = None,
    _headers: Optional[Dict[StrictStr, Any]] = None,
    _host_index: Annotated[StrictInt,
                           Field(ge=0, le=0)] = 0) -> ApiResponse[User]
```

Change user password

Update a user password.

**Arguments**:

- `user_change_password_request` (`UserChangePasswordRequest`): 
- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `_headers` (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `UserChangePasswordRequest`0 (`UserChangePasswordRequest`1): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### user\_update\_password\_without\_preload\_content

```python
@validate_call
def user_update_password_without_preload_content(
    user_change_password_request: Optional[UserChangePasswordRequest] = None,
    _request_timeout: Union[
        None,
        Annotated[StrictFloat, Field(gt=0)],
        Tuple[Annotated[StrictFloat, Field(gt=0)], Annotated[StrictFloat,
                                                             Field(gt=0)]],
    ] = None,
    _request_auth: Optional[Dict[StrictStr, Any]] = None,
    _content_type: Optional[StrictStr] = None,
    _headers: Optional[Dict[StrictStr, Any]] = None,
    _host_index: Annotated[StrictInt,
                           Field(ge=0, le=0)] = 0) -> RESTResponseType
```

Change user password

Update a user password.

**Arguments**:

- `user_change_password_request` (`UserChangePasswordRequest`): 
- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `_headers` (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `UserChangePasswordRequest`0 (`UserChangePasswordRequest`1): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### user\_update\_slack\_oauth\_callback

```python
@validate_call
def user_update_slack_oauth_callback(
        _request_timeout: Union[
            None,
            Annotated[StrictFloat, Field(gt=0)],
            Tuple[Annotated[StrictFloat, Field(gt=0)], Annotated[StrictFloat,
                                                                 Field(gt=0)]],
        ] = None,
        _request_auth: Optional[Dict[StrictStr, Any]] = None,
        _content_type: Optional[StrictStr] = None,
        _headers: Optional[Dict[StrictStr, Any]] = None,
        _host_index: Annotated[StrictInt, Field(ge=0, le=0)] = 0) -> None
```

Complete OAuth flow

Completes the OAuth flow

**Arguments**:

- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `_headers` (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `_host_index` (`int, optional`): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### user\_update\_slack\_oauth\_callback\_with\_http\_info

```python
@validate_call
def user_update_slack_oauth_callback_with_http_info(
    _request_timeout: Union[
        None,
        Annotated[StrictFloat, Field(gt=0)],
        Tuple[Annotated[StrictFloat, Field(gt=0)], Annotated[StrictFloat,
                                                             Field(gt=0)]],
    ] = None,
    _request_auth: Optional[Dict[StrictStr, Any]] = None,
    _content_type: Optional[StrictStr] = None,
    _headers: Optional[Dict[StrictStr, Any]] = None,
    _host_index: Annotated[StrictInt,
                           Field(ge=0, le=0)] = 0) -> ApiResponse[None]
```

Complete OAuth flow

Completes the OAuth flow

**Arguments**:

- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `_headers` (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `_host_index` (`int, optional`): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### user\_update\_slack\_oauth\_callback\_without\_preload\_content

```python
@validate_call
def user_update_slack_oauth_callback_without_preload_content(
    _request_timeout: Union[
        None,
        Annotated[StrictFloat, Field(gt=0)],
        Tuple[Annotated[StrictFloat, Field(gt=0)], Annotated[StrictFloat,
                                                             Field(gt=0)]],
    ] = None,
    _request_auth: Optional[Dict[StrictStr, Any]] = None,
    _content_type: Optional[StrictStr] = None,
    _headers: Optional[Dict[StrictStr, Any]] = None,
    _host_index: Annotated[StrictInt,
                           Field(ge=0, le=0)] = 0) -> RESTResponseType
```

Complete OAuth flow

Completes the OAuth flow

**Arguments**:

- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `_headers` (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `_host_index` (`int, optional`): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### user\_update\_slack\_oauth\_start

```python
@validate_call
def user_update_slack_oauth_start(tenant: Annotated[
    str,
    Field(
        min_length=36, strict=True, max_length=36, description="The tenant id"
    ),
],
                                  _request_timeout: Union[
                                      None,
                                      Annotated[StrictFloat,
                                                Field(gt=0)],
                                      Tuple[Annotated[StrictFloat,
                                                      Field(gt=0)],
                                            Annotated[StrictFloat,
                                                      Field(gt=0)]],
                                  ] = None,
                                  _request_auth: Optional[Dict[StrictStr,
                                                               Any]] = None,
                                  _content_type: Optional[StrictStr] = None,
                                  _headers: Optional[Dict[StrictStr,
                                                          Any]] = None,
                                  _host_index: Annotated[StrictInt,
                                                         Field(ge=0, le=0)] = 0
                                  ) -> None
```

Start OAuth flow

Starts the OAuth flow

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `_headers` (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `str`0 (`str`1): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### user\_update\_slack\_oauth\_start\_with\_http\_info

```python
@validate_call
def user_update_slack_oauth_start_with_http_info(
    tenant: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The tenant id"),
    ],
    _request_timeout: Union[
        None,
        Annotated[StrictFloat, Field(gt=0)],
        Tuple[Annotated[StrictFloat, Field(gt=0)], Annotated[StrictFloat,
                                                             Field(gt=0)]],
    ] = None,
    _request_auth: Optional[Dict[StrictStr, Any]] = None,
    _content_type: Optional[StrictStr] = None,
    _headers: Optional[Dict[StrictStr, Any]] = None,
    _host_index: Annotated[StrictInt,
                           Field(ge=0, le=0)] = 0) -> ApiResponse[None]
```

Start OAuth flow

Starts the OAuth flow

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `_headers` (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `str`0 (`str`1): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### user\_update\_slack\_oauth\_start\_without\_preload\_content

```python
@validate_call
def user_update_slack_oauth_start_without_preload_content(
    tenant: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The tenant id"),
    ],
    _request_timeout: Union[
        None,
        Annotated[StrictFloat, Field(gt=0)],
        Tuple[Annotated[StrictFloat, Field(gt=0)], Annotated[StrictFloat,
                                                             Field(gt=0)]],
    ] = None,
    _request_auth: Optional[Dict[StrictStr, Any]] = None,
    _content_type: Optional[StrictStr] = None,
    _headers: Optional[Dict[StrictStr, Any]] = None,
    _host_index: Annotated[StrictInt,
                           Field(ge=0, le=0)] = 0) -> RESTResponseType
```

Start OAuth flow

Starts the OAuth flow

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `_headers` (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `str`0 (`str`1): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

