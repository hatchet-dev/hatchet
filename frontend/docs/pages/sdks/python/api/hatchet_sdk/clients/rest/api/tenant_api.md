# Table of Contents

* [hatchet\_sdk.clients.rest.api.tenant\_api](#hatchet_sdk.clients.rest.api.tenant_api)
  * [TenantApi](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi)
    * [alert\_email\_group\_create](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.alert_email_group_create)
    * [alert\_email\_group\_create\_with\_http\_info](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.alert_email_group_create_with_http_info)
    * [alert\_email\_group\_create\_without\_preload\_content](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.alert_email_group_create_without_preload_content)
    * [alert\_email\_group\_delete](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.alert_email_group_delete)
    * [alert\_email\_group\_delete\_with\_http\_info](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.alert_email_group_delete_with_http_info)
    * [alert\_email\_group\_delete\_without\_preload\_content](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.alert_email_group_delete_without_preload_content)
    * [alert\_email\_group\_list](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.alert_email_group_list)
    * [alert\_email\_group\_list\_with\_http\_info](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.alert_email_group_list_with_http_info)
    * [alert\_email\_group\_list\_without\_preload\_content](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.alert_email_group_list_without_preload_content)
    * [alert\_email\_group\_update](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.alert_email_group_update)
    * [alert\_email\_group\_update\_with\_http\_info](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.alert_email_group_update_with_http_info)
    * [alert\_email\_group\_update\_without\_preload\_content](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.alert_email_group_update_without_preload_content)
    * [tenant\_alerting\_settings\_get](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.tenant_alerting_settings_get)
    * [tenant\_alerting\_settings\_get\_with\_http\_info](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.tenant_alerting_settings_get_with_http_info)
    * [tenant\_alerting\_settings\_get\_without\_preload\_content](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.tenant_alerting_settings_get_without_preload_content)
    * [tenant\_create](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.tenant_create)
    * [tenant\_create\_with\_http\_info](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.tenant_create_with_http_info)
    * [tenant\_create\_without\_preload\_content](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.tenant_create_without_preload_content)
    * [tenant\_get\_step\_run\_queue\_metrics](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.tenant_get_step_run_queue_metrics)
    * [tenant\_get\_step\_run\_queue\_metrics\_with\_http\_info](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.tenant_get_step_run_queue_metrics_with_http_info)
    * [tenant\_get\_step\_run\_queue\_metrics\_without\_preload\_content](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.tenant_get_step_run_queue_metrics_without_preload_content)
    * [tenant\_invite\_accept](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.tenant_invite_accept)
    * [tenant\_invite\_accept\_with\_http\_info](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.tenant_invite_accept_with_http_info)
    * [tenant\_invite\_accept\_without\_preload\_content](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.tenant_invite_accept_without_preload_content)
    * [tenant\_invite\_create](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.tenant_invite_create)
    * [tenant\_invite\_create\_with\_http\_info](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.tenant_invite_create_with_http_info)
    * [tenant\_invite\_create\_without\_preload\_content](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.tenant_invite_create_without_preload_content)
    * [tenant\_invite\_list](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.tenant_invite_list)
    * [tenant\_invite\_list\_with\_http\_info](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.tenant_invite_list_with_http_info)
    * [tenant\_invite\_list\_without\_preload\_content](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.tenant_invite_list_without_preload_content)
    * [tenant\_invite\_reject](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.tenant_invite_reject)
    * [tenant\_invite\_reject\_with\_http\_info](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.tenant_invite_reject_with_http_info)
    * [tenant\_invite\_reject\_without\_preload\_content](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.tenant_invite_reject_without_preload_content)
    * [tenant\_member\_delete](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.tenant_member_delete)
    * [tenant\_member\_delete\_with\_http\_info](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.tenant_member_delete_with_http_info)
    * [tenant\_member\_delete\_without\_preload\_content](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.tenant_member_delete_without_preload_content)
    * [tenant\_member\_list](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.tenant_member_list)
    * [tenant\_member\_list\_with\_http\_info](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.tenant_member_list_with_http_info)
    * [tenant\_member\_list\_without\_preload\_content](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.tenant_member_list_without_preload_content)
    * [tenant\_resource\_policy\_get](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.tenant_resource_policy_get)
    * [tenant\_resource\_policy\_get\_with\_http\_info](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.tenant_resource_policy_get_with_http_info)
    * [tenant\_resource\_policy\_get\_without\_preload\_content](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.tenant_resource_policy_get_without_preload_content)
    * [tenant\_update](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.tenant_update)
    * [tenant\_update\_with\_http\_info](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.tenant_update_with_http_info)
    * [tenant\_update\_without\_preload\_content](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.tenant_update_without_preload_content)
    * [user\_list\_tenant\_invites](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.user_list_tenant_invites)
    * [user\_list\_tenant\_invites\_with\_http\_info](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.user_list_tenant_invites_with_http_info)
    * [user\_list\_tenant\_invites\_without\_preload\_content](#hatchet_sdk.clients.rest.api.tenant_api.TenantApi.user_list_tenant_invites_without_preload_content)

---
sidebar_label: tenant_api
title: hatchet_sdk.clients.rest.api.tenant_api
---

Hatchet API

The Hatchet API

The version of the OpenAPI document: 1.0.0
Generated by OpenAPI Generator (https://openapi-generator.tech)

Do not edit the class manually.

## TenantApi

NOTE: This class is auto generated by OpenAPI Generator
Ref: https://openapi-generator.tech

Do not edit the class manually.

#### alert\_email\_group\_create

```python
@validate_call
def alert_email_group_create(
    tenant: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The tenant id"),
    ],
    create_tenant_alert_email_group_request: Annotated[
        CreateTenantAlertEmailGroupRequest,
        Field(description="The tenant alert email group to create"),
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
    _host_index: Annotated[StrictInt, Field(ge=0, le=0)] = 0
) -> TenantAlertEmailGroup
```

Create tenant alert email group

Creates a new tenant alert email group

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `create_tenant_alert_email_group_request` (`CreateTenantAlertEmailGroupRequest`): The tenant alert email group to create (required)
- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `str`0 (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `str`2 (`str`3): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### alert\_email\_group\_create\_with\_http\_info

```python
@validate_call
def alert_email_group_create_with_http_info(
    tenant: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The tenant id"),
    ],
    create_tenant_alert_email_group_request: Annotated[
        CreateTenantAlertEmailGroupRequest,
        Field(description="The tenant alert email group to create"),
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
    _host_index: Annotated[StrictInt, Field(ge=0, le=0)] = 0
) -> ApiResponse[TenantAlertEmailGroup]
```

Create tenant alert email group

Creates a new tenant alert email group

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `create_tenant_alert_email_group_request` (`CreateTenantAlertEmailGroupRequest`): The tenant alert email group to create (required)
- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `str`0 (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `str`2 (`str`3): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### alert\_email\_group\_create\_without\_preload\_content

```python
@validate_call
def alert_email_group_create_without_preload_content(
    tenant: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The tenant id"),
    ],
    create_tenant_alert_email_group_request: Annotated[
        CreateTenantAlertEmailGroupRequest,
        Field(description="The tenant alert email group to create"),
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

Create tenant alert email group

Creates a new tenant alert email group

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `create_tenant_alert_email_group_request` (`CreateTenantAlertEmailGroupRequest`): The tenant alert email group to create (required)
- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `str`0 (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `str`2 (`str`3): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### alert\_email\_group\_delete

```python
@validate_call
def alert_email_group_delete(
        alert_email_group: Annotated[
            str,
            Field(
                min_length=36,
                strict=True,
                max_length=36,
                description="The tenant alert email group id",
            ),
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
        _host_index: Annotated[StrictInt, Field(ge=0, le=0)] = 0) -> None
```

Delete tenant alert email group

Deletes a tenant alert email group

**Arguments**:

- `alert_email_group` (`str`): The tenant alert email group id (required)
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

#### alert\_email\_group\_delete\_with\_http\_info

```python
@validate_call
def alert_email_group_delete_with_http_info(
    alert_email_group: Annotated[
        str,
        Field(
            min_length=36,
            strict=True,
            max_length=36,
            description="The tenant alert email group id",
        ),
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

Delete tenant alert email group

Deletes a tenant alert email group

**Arguments**:

- `alert_email_group` (`str`): The tenant alert email group id (required)
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

#### alert\_email\_group\_delete\_without\_preload\_content

```python
@validate_call
def alert_email_group_delete_without_preload_content(
    alert_email_group: Annotated[
        str,
        Field(
            min_length=36,
            strict=True,
            max_length=36,
            description="The tenant alert email group id",
        ),
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

Delete tenant alert email group

Deletes a tenant alert email group

**Arguments**:

- `alert_email_group` (`str`): The tenant alert email group id (required)
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

#### alert\_email\_group\_list

```python
@validate_call
def alert_email_group_list(tenant: Annotated[
    str,
    Field(
        min_length=36, strict=True, max_length=36, description="The tenant id"
    ),
],
                           _request_timeout: Union[
                               None,
                               Annotated[StrictFloat, Field(gt=0)],
                               Tuple[Annotated[StrictFloat,
                                               Field(gt=0)],
                                     Annotated[StrictFloat,
                                               Field(gt=0)]],
                           ] = None,
                           _request_auth: Optional[Dict[StrictStr,
                                                        Any]] = None,
                           _content_type: Optional[StrictStr] = None,
                           _headers: Optional[Dict[StrictStr, Any]] = None,
                           _host_index: Annotated[StrictInt,
                                                  Field(ge=0, le=0)] = 0
                           ) -> TenantAlertEmailGroupList
```

List tenant alert email groups

Gets a list of tenant alert email groups

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

#### alert\_email\_group\_list\_with\_http\_info

```python
@validate_call
def alert_email_group_list_with_http_info(
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
    _host_index: Annotated[StrictInt, Field(ge=0, le=0)] = 0
) -> ApiResponse[TenantAlertEmailGroupList]
```

List tenant alert email groups

Gets a list of tenant alert email groups

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

#### alert\_email\_group\_list\_without\_preload\_content

```python
@validate_call
def alert_email_group_list_without_preload_content(
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

List tenant alert email groups

Gets a list of tenant alert email groups

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

#### alert\_email\_group\_update

```python
@validate_call
def alert_email_group_update(
    alert_email_group: Annotated[
        str,
        Field(
            min_length=36,
            strict=True,
            max_length=36,
            description="The tenant alert email group id",
        ),
    ],
    update_tenant_alert_email_group_request: Annotated[
        UpdateTenantAlertEmailGroupRequest,
        Field(description="The tenant alert email group to update"),
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
    _host_index: Annotated[StrictInt, Field(ge=0, le=0)] = 0
) -> TenantAlertEmailGroup
```

Update tenant alert email group

Updates a tenant alert email group

**Arguments**:

- `alert_email_group` (`str`): The tenant alert email group id (required)
- `update_tenant_alert_email_group_request` (`UpdateTenantAlertEmailGroupRequest`): The tenant alert email group to update (required)
- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `str`0 (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `str`2 (`str`3): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### alert\_email\_group\_update\_with\_http\_info

```python
@validate_call
def alert_email_group_update_with_http_info(
    alert_email_group: Annotated[
        str,
        Field(
            min_length=36,
            strict=True,
            max_length=36,
            description="The tenant alert email group id",
        ),
    ],
    update_tenant_alert_email_group_request: Annotated[
        UpdateTenantAlertEmailGroupRequest,
        Field(description="The tenant alert email group to update"),
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
    _host_index: Annotated[StrictInt, Field(ge=0, le=0)] = 0
) -> ApiResponse[TenantAlertEmailGroup]
```

Update tenant alert email group

Updates a tenant alert email group

**Arguments**:

- `alert_email_group` (`str`): The tenant alert email group id (required)
- `update_tenant_alert_email_group_request` (`UpdateTenantAlertEmailGroupRequest`): The tenant alert email group to update (required)
- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `str`0 (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `str`2 (`str`3): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### alert\_email\_group\_update\_without\_preload\_content

```python
@validate_call
def alert_email_group_update_without_preload_content(
    alert_email_group: Annotated[
        str,
        Field(
            min_length=36,
            strict=True,
            max_length=36,
            description="The tenant alert email group id",
        ),
    ],
    update_tenant_alert_email_group_request: Annotated[
        UpdateTenantAlertEmailGroupRequest,
        Field(description="The tenant alert email group to update"),
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

Update tenant alert email group

Updates a tenant alert email group

**Arguments**:

- `alert_email_group` (`str`): The tenant alert email group id (required)
- `update_tenant_alert_email_group_request` (`UpdateTenantAlertEmailGroupRequest`): The tenant alert email group to update (required)
- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `str`0 (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `str`2 (`str`3): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### tenant\_alerting\_settings\_get

```python
@validate_call
def tenant_alerting_settings_get(tenant: Annotated[
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
                                 ) -> TenantAlertingSettings
```

Get tenant alerting settings

Gets the alerting settings for a tenant

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

#### tenant\_alerting\_settings\_get\_with\_http\_info

```python
@validate_call
def tenant_alerting_settings_get_with_http_info(
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
    _host_index: Annotated[StrictInt, Field(ge=0, le=0)] = 0
) -> ApiResponse[TenantAlertingSettings]
```

Get tenant alerting settings

Gets the alerting settings for a tenant

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

#### tenant\_alerting\_settings\_get\_without\_preload\_content

```python
@validate_call
def tenant_alerting_settings_get_without_preload_content(
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

Get tenant alerting settings

Gets the alerting settings for a tenant

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

#### tenant\_create

```python
@validate_call
def tenant_create(
        create_tenant_request: Annotated[
            CreateTenantRequest,
            Field(description="The tenant to create")],
        _request_timeout: Union[
            None,
            Annotated[StrictFloat, Field(gt=0)],
            Tuple[Annotated[StrictFloat, Field(gt=0)], Annotated[StrictFloat,
                                                                 Field(gt=0)]],
        ] = None,
        _request_auth: Optional[Dict[StrictStr, Any]] = None,
        _content_type: Optional[StrictStr] = None,
        _headers: Optional[Dict[StrictStr, Any]] = None,
        _host_index: Annotated[StrictInt, Field(ge=0, le=0)] = 0) -> Tenant
```

Create tenant

Creates a new tenant

**Arguments**:

- `create_tenant_request` (`CreateTenantRequest`): The tenant to create (required)
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
- `CreateTenantRequest`0 (`CreateTenantRequest`1): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### tenant\_create\_with\_http\_info

```python
@validate_call
def tenant_create_with_http_info(
    create_tenant_request: Annotated[CreateTenantRequest,
                                     Field(
                                         description="The tenant to create")],
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
) -> ApiResponse[Tenant]
```

Create tenant

Creates a new tenant

**Arguments**:

- `create_tenant_request` (`CreateTenantRequest`): The tenant to create (required)
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
- `CreateTenantRequest`0 (`CreateTenantRequest`1): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### tenant\_create\_without\_preload\_content

```python
@validate_call
def tenant_create_without_preload_content(
    create_tenant_request: Annotated[CreateTenantRequest,
                                     Field(
                                         description="The tenant to create")],
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

Create tenant

Creates a new tenant

**Arguments**:

- `create_tenant_request` (`CreateTenantRequest`): The tenant to create (required)
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
- `CreateTenantRequest`0 (`CreateTenantRequest`1): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### tenant\_get\_step\_run\_queue\_metrics

```python
@validate_call
def tenant_get_step_run_queue_metrics(
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
    _host_index: Annotated[StrictInt, Field(ge=0, le=0)] = 0
) -> TenantStepRunQueueMetrics
```

Get step run metrics

Get the queue metrics for the tenant

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

#### tenant\_get\_step\_run\_queue\_metrics\_with\_http\_info

```python
@validate_call
def tenant_get_step_run_queue_metrics_with_http_info(
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
    _host_index: Annotated[StrictInt, Field(ge=0, le=0)] = 0
) -> ApiResponse[TenantStepRunQueueMetrics]
```

Get step run metrics

Get the queue metrics for the tenant

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

#### tenant\_get\_step\_run\_queue\_metrics\_without\_preload\_content

```python
@validate_call
def tenant_get_step_run_queue_metrics_without_preload_content(
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

Get step run metrics

Get the queue metrics for the tenant

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

#### tenant\_invite\_accept

```python
@validate_call
def tenant_invite_accept(
        accept_invite_request: Optional[AcceptInviteRequest] = None,
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

Accept tenant invite

Accepts a tenant invite

**Arguments**:

- `accept_invite_request` (`AcceptInviteRequest`): 
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
- `AcceptInviteRequest`0 (`AcceptInviteRequest`1): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### tenant\_invite\_accept\_with\_http\_info

```python
@validate_call
def tenant_invite_accept_with_http_info(
    accept_invite_request: Optional[AcceptInviteRequest] = None,
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

Accept tenant invite

Accepts a tenant invite

**Arguments**:

- `accept_invite_request` (`AcceptInviteRequest`): 
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
- `AcceptInviteRequest`0 (`AcceptInviteRequest`1): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### tenant\_invite\_accept\_without\_preload\_content

```python
@validate_call
def tenant_invite_accept_without_preload_content(
    accept_invite_request: Optional[AcceptInviteRequest] = None,
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

Accept tenant invite

Accepts a tenant invite

**Arguments**:

- `accept_invite_request` (`AcceptInviteRequest`): 
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
- `AcceptInviteRequest`0 (`AcceptInviteRequest`1): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### tenant\_invite\_create

```python
@validate_call
def tenant_invite_create(tenant: Annotated[
    str,
    Field(
        min_length=36, strict=True, max_length=36, description="The tenant id"
    ),
],
                         create_tenant_invite_request: Annotated[
                             CreateTenantInviteRequest,
                             Field(description="The tenant invite to create")],
                         _request_timeout: Union[
                             None,
                             Annotated[StrictFloat, Field(gt=0)],
                             Tuple[Annotated[StrictFloat,
                                             Field(gt=0)],
                                   Annotated[StrictFloat,
                                             Field(gt=0)]],
                         ] = None,
                         _request_auth: Optional[Dict[StrictStr, Any]] = None,
                         _content_type: Optional[StrictStr] = None,
                         _headers: Optional[Dict[StrictStr, Any]] = None,
                         _host_index: Annotated[StrictInt,
                                                Field(ge=0, le=0)] = 0
                         ) -> TenantInvite
```

Create tenant invite

Creates a new tenant invite

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `create_tenant_invite_request` (`CreateTenantInviteRequest`): The tenant invite to create (required)
- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `str`0 (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `str`2 (`str`3): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### tenant\_invite\_create\_with\_http\_info

```python
@validate_call
def tenant_invite_create_with_http_info(
    tenant: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The tenant id"),
    ],
    create_tenant_invite_request: Annotated[
        CreateTenantInviteRequest,
        Field(description="The tenant invite to create")],
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
) -> ApiResponse[TenantInvite]
```

Create tenant invite

Creates a new tenant invite

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `create_tenant_invite_request` (`CreateTenantInviteRequest`): The tenant invite to create (required)
- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `str`0 (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `str`2 (`str`3): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### tenant\_invite\_create\_without\_preload\_content

```python
@validate_call
def tenant_invite_create_without_preload_content(
    tenant: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The tenant id"),
    ],
    create_tenant_invite_request: Annotated[
        CreateTenantInviteRequest,
        Field(description="The tenant invite to create")],
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

Create tenant invite

Creates a new tenant invite

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `create_tenant_invite_request` (`CreateTenantInviteRequest`): The tenant invite to create (required)
- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `str`0 (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `str`2 (`str`3): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### tenant\_invite\_list

```python
@validate_call
def tenant_invite_list(tenant: Annotated[
    str,
    Field(
        min_length=36, strict=True, max_length=36, description="The tenant id"
    ),
],
                       _request_timeout: Union[
                           None,
                           Annotated[StrictFloat, Field(gt=0)],
                           Tuple[Annotated[StrictFloat,
                                           Field(gt=0)],
                                 Annotated[StrictFloat,
                                           Field(gt=0)]],
                       ] = None,
                       _request_auth: Optional[Dict[StrictStr, Any]] = None,
                       _content_type: Optional[StrictStr] = None,
                       _headers: Optional[Dict[StrictStr, Any]] = None,
                       _host_index: Annotated[StrictInt,
                                              Field(ge=0, le=0)] = 0
                       ) -> TenantInviteList
```

List tenant invites

Gets a list of tenant invites

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

#### tenant\_invite\_list\_with\_http\_info

```python
@validate_call
def tenant_invite_list_with_http_info(
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
    _host_index: Annotated[StrictInt, Field(ge=0, le=0)] = 0
) -> ApiResponse[TenantInviteList]
```

List tenant invites

Gets a list of tenant invites

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

#### tenant\_invite\_list\_without\_preload\_content

```python
@validate_call
def tenant_invite_list_without_preload_content(
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

List tenant invites

Gets a list of tenant invites

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

#### tenant\_invite\_reject

```python
@validate_call
def tenant_invite_reject(
        reject_invite_request: Optional[RejectInviteRequest] = None,
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

Reject tenant invite

Rejects a tenant invite

**Arguments**:

- `reject_invite_request` (`RejectInviteRequest`): 
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
- `RejectInviteRequest`0 (`RejectInviteRequest`1): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### tenant\_invite\_reject\_with\_http\_info

```python
@validate_call
def tenant_invite_reject_with_http_info(
    reject_invite_request: Optional[RejectInviteRequest] = None,
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

Reject tenant invite

Rejects a tenant invite

**Arguments**:

- `reject_invite_request` (`RejectInviteRequest`): 
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
- `RejectInviteRequest`0 (`RejectInviteRequest`1): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### tenant\_invite\_reject\_without\_preload\_content

```python
@validate_call
def tenant_invite_reject_without_preload_content(
    reject_invite_request: Optional[RejectInviteRequest] = None,
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

Reject tenant invite

Rejects a tenant invite

**Arguments**:

- `reject_invite_request` (`RejectInviteRequest`): 
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
- `RejectInviteRequest`0 (`RejectInviteRequest`1): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### tenant\_member\_delete

```python
@validate_call
def tenant_member_delete(tenant: Annotated[
    str,
    Field(
        min_length=36, strict=True, max_length=36, description="The tenant id"
    ),
],
                         member: Annotated[
                             str,
                             Field(
                                 min_length=36,
                                 strict=True,
                                 max_length=36,
                                 description="The tenant member id",
                             ),
                         ],
                         _request_timeout: Union[
                             None,
                             Annotated[StrictFloat, Field(gt=0)],
                             Tuple[Annotated[StrictFloat,
                                             Field(gt=0)],
                                   Annotated[StrictFloat,
                                             Field(gt=0)]],
                         ] = None,
                         _request_auth: Optional[Dict[StrictStr, Any]] = None,
                         _content_type: Optional[StrictStr] = None,
                         _headers: Optional[Dict[StrictStr, Any]] = None,
                         _host_index: Annotated[StrictInt,
                                                Field(ge=0, le=0)] = 0
                         ) -> TenantMember
```

Delete a tenant member

Delete a member from a tenant

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `member` (`str`): The tenant member id (required)
- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `str`0 (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `str`2 (`str`3): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### tenant\_member\_delete\_with\_http\_info

```python
@validate_call
def tenant_member_delete_with_http_info(
    tenant: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The tenant id"),
    ],
    member: Annotated[
        str,
        Field(
            min_length=36,
            strict=True,
            max_length=36,
            description="The tenant member id",
        ),
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
    _host_index: Annotated[StrictInt, Field(ge=0, le=0)] = 0
) -> ApiResponse[TenantMember]
```

Delete a tenant member

Delete a member from a tenant

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `member` (`str`): The tenant member id (required)
- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `str`0 (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `str`2 (`str`3): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### tenant\_member\_delete\_without\_preload\_content

```python
@validate_call
def tenant_member_delete_without_preload_content(
    tenant: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The tenant id"),
    ],
    member: Annotated[
        str,
        Field(
            min_length=36,
            strict=True,
            max_length=36,
            description="The tenant member id",
        ),
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

Delete a tenant member

Delete a member from a tenant

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `member` (`str`): The tenant member id (required)
- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `str`0 (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `str`2 (`str`3): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### tenant\_member\_list

```python
@validate_call
def tenant_member_list(tenant: Annotated[
    str,
    Field(
        min_length=36, strict=True, max_length=36, description="The tenant id"
    ),
],
                       _request_timeout: Union[
                           None,
                           Annotated[StrictFloat, Field(gt=0)],
                           Tuple[Annotated[StrictFloat,
                                           Field(gt=0)],
                                 Annotated[StrictFloat,
                                           Field(gt=0)]],
                       ] = None,
                       _request_auth: Optional[Dict[StrictStr, Any]] = None,
                       _content_type: Optional[StrictStr] = None,
                       _headers: Optional[Dict[StrictStr, Any]] = None,
                       _host_index: Annotated[StrictInt,
                                              Field(ge=0, le=0)] = 0
                       ) -> TenantMemberList
```

List tenant members

Gets a list of tenant members

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

#### tenant\_member\_list\_with\_http\_info

```python
@validate_call
def tenant_member_list_with_http_info(
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
    _host_index: Annotated[StrictInt, Field(ge=0, le=0)] = 0
) -> ApiResponse[TenantMemberList]
```

List tenant members

Gets a list of tenant members

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

#### tenant\_member\_list\_without\_preload\_content

```python
@validate_call
def tenant_member_list_without_preload_content(
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

List tenant members

Gets a list of tenant members

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

#### tenant\_resource\_policy\_get

```python
@validate_call
def tenant_resource_policy_get(tenant: Annotated[
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
                               _headers: Optional[Dict[StrictStr, Any]] = None,
                               _host_index: Annotated[StrictInt,
                                                      Field(ge=0, le=0)] = 0
                               ) -> TenantResourcePolicy
```

Create tenant alert email group

Gets the resource policy for a tenant

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

#### tenant\_resource\_policy\_get\_with\_http\_info

```python
@validate_call
def tenant_resource_policy_get_with_http_info(
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
    _host_index: Annotated[StrictInt, Field(ge=0, le=0)] = 0
) -> ApiResponse[TenantResourcePolicy]
```

Create tenant alert email group

Gets the resource policy for a tenant

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

#### tenant\_resource\_policy\_get\_without\_preload\_content

```python
@validate_call
def tenant_resource_policy_get_without_preload_content(
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

Create tenant alert email group

Gets the resource policy for a tenant

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

#### tenant\_update

```python
@validate_call
def tenant_update(tenant: Annotated[
    str,
    Field(
        min_length=36, strict=True, max_length=36, description="The tenant id"
    ),
],
                  update_tenant_request: Annotated[
                      UpdateTenantRequest,
                      Field(description="The tenant properties to update")],
                  _request_timeout: Union[
                      None,
                      Annotated[StrictFloat, Field(gt=0)],
                      Tuple[Annotated[StrictFloat, Field(gt=0)],
                            Annotated[StrictFloat, Field(gt=0)]],
                  ] = None,
                  _request_auth: Optional[Dict[StrictStr, Any]] = None,
                  _content_type: Optional[StrictStr] = None,
                  _headers: Optional[Dict[StrictStr, Any]] = None,
                  _host_index: Annotated[StrictInt,
                                         Field(ge=0, le=0)] = 0) -> Tenant
```

Update tenant

Update an existing tenant

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `update_tenant_request` (`UpdateTenantRequest`): The tenant properties to update (required)
- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `str`0 (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `str`2 (`str`3): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### tenant\_update\_with\_http\_info

```python
@validate_call
def tenant_update_with_http_info(
    tenant: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The tenant id"),
    ],
    update_tenant_request: Annotated[
        UpdateTenantRequest,
        Field(description="The tenant properties to update")],
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
) -> ApiResponse[Tenant]
```

Update tenant

Update an existing tenant

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `update_tenant_request` (`UpdateTenantRequest`): The tenant properties to update (required)
- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `str`0 (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `str`2 (`str`3): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### tenant\_update\_without\_preload\_content

```python
@validate_call
def tenant_update_without_preload_content(
    tenant: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The tenant id"),
    ],
    update_tenant_request: Annotated[
        UpdateTenantRequest,
        Field(description="The tenant properties to update")],
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

Update tenant

Update an existing tenant

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `update_tenant_request` (`UpdateTenantRequest`): The tenant properties to update (required)
- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `_content_type` (`str, Optional`): force content-type for the request.
- `str`0 (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `str`2 (`str`3): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### user\_list\_tenant\_invites

```python
@validate_call
def user_list_tenant_invites(
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
                           Field(ge=0, le=0)] = 0) -> TenantInviteList
```

List tenant invites

Lists all tenant invites for the current user

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

#### user\_list\_tenant\_invites\_with\_http\_info

```python
@validate_call
def user_list_tenant_invites_with_http_info(
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
) -> ApiResponse[TenantInviteList]
```

List tenant invites

Lists all tenant invites for the current user

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

#### user\_list\_tenant\_invites\_without\_preload\_content

```python
@validate_call
def user_list_tenant_invites_without_preload_content(
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

List tenant invites

Lists all tenant invites for the current user

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

