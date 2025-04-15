# Table of Contents

* [hatchet\_sdk.v0.clients.rest.api.workflow\_api](#hatchet_sdk.v0.clients.rest.api.workflow_api)
  * [WorkflowApi](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi)
    * [cron\_workflow\_list](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.cron_workflow_list)
    * [cron\_workflow\_list\_with\_http\_info](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.cron_workflow_list_with_http_info)
    * [cron\_workflow\_list\_without\_preload\_content](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.cron_workflow_list_without_preload_content)
    * [tenant\_get\_queue\_metrics](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.tenant_get_queue_metrics)
    * [tenant\_get\_queue\_metrics\_with\_http\_info](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.tenant_get_queue_metrics_with_http_info)
    * [tenant\_get\_queue\_metrics\_without\_preload\_content](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.tenant_get_queue_metrics_without_preload_content)
    * [workflow\_cron\_delete](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_cron_delete)
    * [workflow\_cron\_delete\_with\_http\_info](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_cron_delete_with_http_info)
    * [workflow\_cron\_delete\_without\_preload\_content](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_cron_delete_without_preload_content)
    * [workflow\_cron\_get](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_cron_get)
    * [workflow\_cron\_get\_with\_http\_info](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_cron_get_with_http_info)
    * [workflow\_cron\_get\_without\_preload\_content](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_cron_get_without_preload_content)
    * [workflow\_delete](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_delete)
    * [workflow\_delete\_with\_http\_info](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_delete_with_http_info)
    * [workflow\_delete\_without\_preload\_content](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_delete_without_preload_content)
    * [workflow\_get](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_get)
    * [workflow\_get\_with\_http\_info](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_get_with_http_info)
    * [workflow\_get\_without\_preload\_content](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_get_without_preload_content)
    * [workflow\_get\_metrics](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_get_metrics)
    * [workflow\_get\_metrics\_with\_http\_info](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_get_metrics_with_http_info)
    * [workflow\_get\_metrics\_without\_preload\_content](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_get_metrics_without_preload_content)
    * [workflow\_get\_workers\_count](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_get_workers_count)
    * [workflow\_get\_workers\_count\_with\_http\_info](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_get_workers_count_with_http_info)
    * [workflow\_get\_workers\_count\_without\_preload\_content](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_get_workers_count_without_preload_content)
    * [workflow\_list](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_list)
    * [workflow\_list\_with\_http\_info](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_list_with_http_info)
    * [workflow\_list\_without\_preload\_content](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_list_without_preload_content)
    * [workflow\_run\_get](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_run_get)
    * [workflow\_run\_get\_with\_http\_info](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_run_get_with_http_info)
    * [workflow\_run\_get\_without\_preload\_content](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_run_get_without_preload_content)
    * [workflow\_run\_get\_metrics](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_run_get_metrics)
    * [workflow\_run\_get\_metrics\_with\_http\_info](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_run_get_metrics_with_http_info)
    * [workflow\_run\_get\_metrics\_without\_preload\_content](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_run_get_metrics_without_preload_content)
    * [workflow\_run\_get\_shape](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_run_get_shape)
    * [workflow\_run\_get\_shape\_with\_http\_info](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_run_get_shape_with_http_info)
    * [workflow\_run\_get\_shape\_without\_preload\_content](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_run_get_shape_without_preload_content)
    * [workflow\_run\_list](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_run_list)
    * [workflow\_run\_list\_with\_http\_info](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_run_list_with_http_info)
    * [workflow\_run\_list\_without\_preload\_content](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_run_list_without_preload_content)
    * [workflow\_scheduled\_delete](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_scheduled_delete)
    * [workflow\_scheduled\_delete\_with\_http\_info](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_scheduled_delete_with_http_info)
    * [workflow\_scheduled\_delete\_without\_preload\_content](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_scheduled_delete_without_preload_content)
    * [workflow\_scheduled\_get](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_scheduled_get)
    * [workflow\_scheduled\_get\_with\_http\_info](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_scheduled_get_with_http_info)
    * [workflow\_scheduled\_get\_without\_preload\_content](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_scheduled_get_without_preload_content)
    * [workflow\_scheduled\_list](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_scheduled_list)
    * [workflow\_scheduled\_list\_with\_http\_info](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_scheduled_list_with_http_info)
    * [workflow\_scheduled\_list\_without\_preload\_content](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_scheduled_list_without_preload_content)
    * [workflow\_update](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_update)
    * [workflow\_update\_with\_http\_info](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_update_with_http_info)
    * [workflow\_update\_without\_preload\_content](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_update_without_preload_content)
    * [workflow\_version\_get](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_version_get)
    * [workflow\_version\_get\_with\_http\_info](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_version_get_with_http_info)
    * [workflow\_version\_get\_without\_preload\_content](#hatchet_sdk.v0.clients.rest.api.workflow_api.WorkflowApi.workflow_version_get_without_preload_content)

---
sidebar_label: workflow_api
title: hatchet_sdk.v0.clients.rest.api.workflow_api
---

Hatchet API

The Hatchet API

The version of the OpenAPI document: 1.0.0
Generated by OpenAPI Generator (https://openapi-generator.tech)

Do not edit the class manually.

## WorkflowApi

NOTE: This class is auto generated by OpenAPI Generator
Ref: https://openapi-generator.tech

Do not edit the class manually.

#### cron\_workflow\_list

```python
@validate_call
async def cron_workflow_list(
    tenant: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The tenant id"),
    ],
    offset: Annotated[Optional[StrictInt],
                      Field(description="The number to skip")] = None,
    limit: Annotated[Optional[StrictInt],
                     Field(description="The number to limit by")] = None,
    workflow_id: Annotated[
        Optional[Annotated[str,
                           Field(min_length=36, strict=True, max_length=36)]],
        Field(description="The workflow id to get runs for."),
    ] = None,
    additional_metadata: Annotated[
        Optional[List[StrictStr]],
        Field(description="A list of metadata key value pairs to filter by"),
    ] = None,
    order_by_field: Annotated[Optional[CronWorkflowsOrderByField],
                              Field(description="The order by field")] = None,
    order_by_direction: Annotated[
        Optional[WorkflowRunOrderByDirection],
        Field(description="The order by direction"),
    ] = None,
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
                           Field(ge=0, le=0)] = 0) -> CronWorkflowsList
```

Get cron job workflows

Get all cron job workflow triggers for a tenant

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `offset` (`int`): The number to skip
- `limit` (`int`): The number to limit by
- `workflow_id` (`str`): The workflow id to get runs for.
- `additional_metadata` (`List[str]`): A list of metadata key value pairs to filter by
- `str`0 (`str`1): The order by field
- `str`2 (`str`3): The order by direction
- `str`4 (`str`5): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `str`6 (`str`7): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `str`8 (`str`9): force content-type for the request.
- `offset`0 (`str`7): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `offset`2 (`offset`3): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### cron\_workflow\_list\_with\_http\_info

```python
@validate_call
async def cron_workflow_list_with_http_info(
    tenant: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The tenant id"),
    ],
    offset: Annotated[Optional[StrictInt],
                      Field(description="The number to skip")] = None,
    limit: Annotated[Optional[StrictInt],
                     Field(description="The number to limit by")] = None,
    workflow_id: Annotated[
        Optional[Annotated[str,
                           Field(min_length=36, strict=True, max_length=36)]],
        Field(description="The workflow id to get runs for."),
    ] = None,
    additional_metadata: Annotated[
        Optional[List[StrictStr]],
        Field(description="A list of metadata key value pairs to filter by"),
    ] = None,
    order_by_field: Annotated[Optional[CronWorkflowsOrderByField],
                              Field(description="The order by field")] = None,
    order_by_direction: Annotated[
        Optional[WorkflowRunOrderByDirection],
        Field(description="The order by direction"),
    ] = None,
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
) -> ApiResponse[CronWorkflowsList]
```

Get cron job workflows

Get all cron job workflow triggers for a tenant

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `offset` (`int`): The number to skip
- `limit` (`int`): The number to limit by
- `workflow_id` (`str`): The workflow id to get runs for.
- `additional_metadata` (`List[str]`): A list of metadata key value pairs to filter by
- `str`0 (`str`1): The order by field
- `str`2 (`str`3): The order by direction
- `str`4 (`str`5): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `str`6 (`str`7): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `str`8 (`str`9): force content-type for the request.
- `offset`0 (`str`7): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `offset`2 (`offset`3): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### cron\_workflow\_list\_without\_preload\_content

```python
@validate_call
async def cron_workflow_list_without_preload_content(
    tenant: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The tenant id"),
    ],
    offset: Annotated[Optional[StrictInt],
                      Field(description="The number to skip")] = None,
    limit: Annotated[Optional[StrictInt],
                     Field(description="The number to limit by")] = None,
    workflow_id: Annotated[
        Optional[Annotated[str,
                           Field(min_length=36, strict=True, max_length=36)]],
        Field(description="The workflow id to get runs for."),
    ] = None,
    additional_metadata: Annotated[
        Optional[List[StrictStr]],
        Field(description="A list of metadata key value pairs to filter by"),
    ] = None,
    order_by_field: Annotated[Optional[CronWorkflowsOrderByField],
                              Field(description="The order by field")] = None,
    order_by_direction: Annotated[
        Optional[WorkflowRunOrderByDirection],
        Field(description="The order by direction"),
    ] = None,
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

Get cron job workflows

Get all cron job workflow triggers for a tenant

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `offset` (`int`): The number to skip
- `limit` (`int`): The number to limit by
- `workflow_id` (`str`): The workflow id to get runs for.
- `additional_metadata` (`List[str]`): A list of metadata key value pairs to filter by
- `str`0 (`str`1): The order by field
- `str`2 (`str`3): The order by direction
- `str`4 (`str`5): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `str`6 (`str`7): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `str`8 (`str`9): force content-type for the request.
- `offset`0 (`str`7): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `offset`2 (`offset`3): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### tenant\_get\_queue\_metrics

```python
@validate_call
async def tenant_get_queue_metrics(
    tenant: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The tenant id"),
    ],
    workflows: Annotated[
        Optional[List[StrictStr]],
        Field(description="A list of workflow IDs to filter by"),
    ] = None,
    additional_metadata: Annotated[
        Optional[List[StrictStr]],
        Field(description="A list of metadata key value pairs to filter by"),
    ] = None,
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
) -> TenantQueueMetrics
```

Get workflow metrics

Get the queue metrics for the tenant

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `workflows` (`List[str]`): A list of workflow IDs to filter by
- `additional_metadata` (`List[str]`): A list of metadata key value pairs to filter by
- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `str`0 (`str`1): force content-type for the request.
- `str`2 (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `str`4 (`str`5): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### tenant\_get\_queue\_metrics\_with\_http\_info

```python
@validate_call
async def tenant_get_queue_metrics_with_http_info(
    tenant: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The tenant id"),
    ],
    workflows: Annotated[
        Optional[List[StrictStr]],
        Field(description="A list of workflow IDs to filter by"),
    ] = None,
    additional_metadata: Annotated[
        Optional[List[StrictStr]],
        Field(description="A list of metadata key value pairs to filter by"),
    ] = None,
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
) -> ApiResponse[TenantQueueMetrics]
```

Get workflow metrics

Get the queue metrics for the tenant

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `workflows` (`List[str]`): A list of workflow IDs to filter by
- `additional_metadata` (`List[str]`): A list of metadata key value pairs to filter by
- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `str`0 (`str`1): force content-type for the request.
- `str`2 (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `str`4 (`str`5): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### tenant\_get\_queue\_metrics\_without\_preload\_content

```python
@validate_call
async def tenant_get_queue_metrics_without_preload_content(
    tenant: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The tenant id"),
    ],
    workflows: Annotated[
        Optional[List[StrictStr]],
        Field(description="A list of workflow IDs to filter by"),
    ] = None,
    additional_metadata: Annotated[
        Optional[List[StrictStr]],
        Field(description="A list of metadata key value pairs to filter by"),
    ] = None,
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

Get workflow metrics

Get the queue metrics for the tenant

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `workflows` (`List[str]`): A list of workflow IDs to filter by
- `additional_metadata` (`List[str]`): A list of metadata key value pairs to filter by
- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `str`0 (`str`1): force content-type for the request.
- `str`2 (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `str`4 (`str`5): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### workflow\_cron\_delete

```python
@validate_call
async def workflow_cron_delete(tenant: Annotated[
    str,
    Field(
        min_length=36, strict=True, max_length=36, description="The tenant id"
    ),
],
                               cron_workflow: Annotated[
                                   str,
                                   Field(min_length=36,
                                         strict=True,
                                         max_length=36,
                                         description="The cron job id"),
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
                               ) -> None
```

Delete cron job workflow run

Delete a cron job workflow run for a tenant

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `cron_workflow` (`str`): The cron job id (required)
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

#### workflow\_cron\_delete\_with\_http\_info

```python
@validate_call
async def workflow_cron_delete_with_http_info(
    tenant: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The tenant id"),
    ],
    cron_workflow: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The cron job id"),
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

Delete cron job workflow run

Delete a cron job workflow run for a tenant

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `cron_workflow` (`str`): The cron job id (required)
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

#### workflow\_cron\_delete\_without\_preload\_content

```python
@validate_call
async def workflow_cron_delete_without_preload_content(
    tenant: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The tenant id"),
    ],
    cron_workflow: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The cron job id"),
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

Delete cron job workflow run

Delete a cron job workflow run for a tenant

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `cron_workflow` (`str`): The cron job id (required)
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

#### workflow\_cron\_get

```python
@validate_call
async def workflow_cron_get(tenant: Annotated[
    str,
    Field(
        min_length=36, strict=True, max_length=36, description="The tenant id"
    ),
],
                            cron_workflow: Annotated[
                                str,
                                Field(min_length=36,
                                      strict=True,
                                      max_length=36,
                                      description="The cron job id"),
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
                            ) -> CronWorkflows
```

Get cron job workflow run

Get a cron job workflow run for a tenant

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `cron_workflow` (`str`): The cron job id (required)
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

#### workflow\_cron\_get\_with\_http\_info

```python
@validate_call
async def workflow_cron_get_with_http_info(
    tenant: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The tenant id"),
    ],
    cron_workflow: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The cron job id"),
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
) -> ApiResponse[CronWorkflows]
```

Get cron job workflow run

Get a cron job workflow run for a tenant

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `cron_workflow` (`str`): The cron job id (required)
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

#### workflow\_cron\_get\_without\_preload\_content

```python
@validate_call
async def workflow_cron_get_without_preload_content(
    tenant: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The tenant id"),
    ],
    cron_workflow: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The cron job id"),
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

Get cron job workflow run

Get a cron job workflow run for a tenant

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `cron_workflow` (`str`): The cron job id (required)
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

#### workflow\_delete

```python
@validate_call
async def workflow_delete(
        workflow: Annotated[
            str,
            Field(min_length=36,
                  strict=True,
                  max_length=36,
                  description="The workflow id"),
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

Delete workflow

Delete a workflow for a tenant

**Arguments**:

- `workflow` (`str`): The workflow id (required)
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

#### workflow\_delete\_with\_http\_info

```python
@validate_call
async def workflow_delete_with_http_info(
    workflow: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The workflow id"),
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

Delete workflow

Delete a workflow for a tenant

**Arguments**:

- `workflow` (`str`): The workflow id (required)
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

#### workflow\_delete\_without\_preload\_content

```python
@validate_call
async def workflow_delete_without_preload_content(
    workflow: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The workflow id"),
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

Delete workflow

Delete a workflow for a tenant

**Arguments**:

- `workflow` (`str`): The workflow id (required)
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

#### workflow\_get

```python
@validate_call
async def workflow_get(
        workflow: Annotated[
            str,
            Field(min_length=36,
                  strict=True,
                  max_length=36,
                  description="The workflow id"),
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
        _host_index: Annotated[StrictInt, Field(ge=0, le=0)] = 0) -> Workflow
```

Get workflow

Get a workflow for a tenant

**Arguments**:

- `workflow` (`str`): The workflow id (required)
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

#### workflow\_get\_with\_http\_info

```python
@validate_call
async def workflow_get_with_http_info(
    workflow: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The workflow id"),
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
) -> ApiResponse[Workflow]
```

Get workflow

Get a workflow for a tenant

**Arguments**:

- `workflow` (`str`): The workflow id (required)
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

#### workflow\_get\_without\_preload\_content

```python
@validate_call
async def workflow_get_without_preload_content(
    workflow: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The workflow id"),
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

Get workflow

Get a workflow for a tenant

**Arguments**:

- `workflow` (`str`): The workflow id (required)
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

#### workflow\_get\_metrics

```python
@validate_call
async def workflow_get_metrics(
    workflow: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The workflow id"),
    ],
    status: Annotated[
        Optional[WorkflowRunStatus],
        Field(description="A status of workflow run statuses to filter by"),
    ] = None,
    group_key: Annotated[
        Optional[StrictStr],
        Field(description="A group key to filter metrics by")] = None,
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
                           Field(ge=0, le=0)] = 0) -> WorkflowMetrics
```

Get workflow metrics

Get the metrics for a workflow version

**Arguments**:

- `workflow` (`str`): The workflow id (required)
- `status` (`WorkflowRunStatus`): A status of workflow run statuses to filter by
- `group_key` (`str`): A group key to filter metrics by
- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `str`0 (`str`1): force content-type for the request.
- `str`2 (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `str`4 (`str`5): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### workflow\_get\_metrics\_with\_http\_info

```python
@validate_call
async def workflow_get_metrics_with_http_info(
    workflow: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The workflow id"),
    ],
    status: Annotated[
        Optional[WorkflowRunStatus],
        Field(description="A status of workflow run statuses to filter by"),
    ] = None,
    group_key: Annotated[
        Optional[StrictStr],
        Field(description="A group key to filter metrics by")] = None,
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
) -> ApiResponse[WorkflowMetrics]
```

Get workflow metrics

Get the metrics for a workflow version

**Arguments**:

- `workflow` (`str`): The workflow id (required)
- `status` (`WorkflowRunStatus`): A status of workflow run statuses to filter by
- `group_key` (`str`): A group key to filter metrics by
- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `str`0 (`str`1): force content-type for the request.
- `str`2 (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `str`4 (`str`5): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### workflow\_get\_metrics\_without\_preload\_content

```python
@validate_call
async def workflow_get_metrics_without_preload_content(
    workflow: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The workflow id"),
    ],
    status: Annotated[
        Optional[WorkflowRunStatus],
        Field(description="A status of workflow run statuses to filter by"),
    ] = None,
    group_key: Annotated[
        Optional[StrictStr],
        Field(description="A group key to filter metrics by")] = None,
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

Get workflow metrics

Get the metrics for a workflow version

**Arguments**:

- `workflow` (`str`): The workflow id (required)
- `status` (`WorkflowRunStatus`): A status of workflow run statuses to filter by
- `group_key` (`str`): A group key to filter metrics by
- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `_request_auth` (`dict, optional`): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `str`0 (`str`1): force content-type for the request.
- `str`2 (`dict, optional`): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `str`4 (`str`5): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### workflow\_get\_workers\_count

```python
@validate_call
async def workflow_get_workers_count(tenant: Annotated[
    str,
    Field(
        min_length=36, strict=True, max_length=36, description="The tenant id"
    ),
],
                                     workflow: Annotated[
                                         str,
                                         Field(min_length=36,
                                               strict=True,
                                               max_length=36,
                                               description="The workflow id"),
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
                                     _host_index: Annotated[
                                         StrictInt,
                                         Field(ge=0, le=0)] = 0
                                     ) -> WorkflowWorkersCount
```

Get workflow worker count

Get a count of the workers available for workflow

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `workflow` (`str`): The workflow id (required)
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

#### workflow\_get\_workers\_count\_with\_http\_info

```python
@validate_call
async def workflow_get_workers_count_with_http_info(
    tenant: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The tenant id"),
    ],
    workflow: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The workflow id"),
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
) -> ApiResponse[WorkflowWorkersCount]
```

Get workflow worker count

Get a count of the workers available for workflow

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `workflow` (`str`): The workflow id (required)
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

#### workflow\_get\_workers\_count\_without\_preload\_content

```python
@validate_call
async def workflow_get_workers_count_without_preload_content(
    tenant: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The tenant id"),
    ],
    workflow: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The workflow id"),
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

Get workflow worker count

Get a count of the workers available for workflow

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `workflow` (`str`): The workflow id (required)
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

#### workflow\_list

```python
@validate_call
async def workflow_list(
        tenant: Annotated[
            str,
            Field(min_length=36,
                  strict=True,
                  max_length=36,
                  description="The tenant id"),
        ],
        offset: Annotated[Optional[StrictInt],
                          Field(description="The number to skip")] = None,
        limit: Annotated[Optional[StrictInt],
                         Field(description="The number to limit by")] = None,
        name: Annotated[Optional[StrictStr],
                        Field(description="Search by name")] = None,
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
                               Field(ge=0, le=0)] = 0) -> WorkflowList
```

Get workflows

Get all workflows for a tenant

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `offset` (`int`): The number to skip
- `limit` (`int`): The number to limit by
- `name` (`str`): Search by name
- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `str`0 (`str`1): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `str`2 (`str`3): force content-type for the request.
- `str`4 (`str`1): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `str`6 (`str`7): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### workflow\_list\_with\_http\_info

```python
@validate_call
async def workflow_list_with_http_info(
    tenant: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The tenant id"),
    ],
    offset: Annotated[Optional[StrictInt],
                      Field(description="The number to skip")] = None,
    limit: Annotated[Optional[StrictInt],
                     Field(description="The number to limit by")] = None,
    name: Annotated[Optional[StrictStr],
                    Field(description="Search by name")] = None,
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
) -> ApiResponse[WorkflowList]
```

Get workflows

Get all workflows for a tenant

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `offset` (`int`): The number to skip
- `limit` (`int`): The number to limit by
- `name` (`str`): Search by name
- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `str`0 (`str`1): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `str`2 (`str`3): force content-type for the request.
- `str`4 (`str`1): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `str`6 (`str`7): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### workflow\_list\_without\_preload\_content

```python
@validate_call
async def workflow_list_without_preload_content(
    tenant: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The tenant id"),
    ],
    offset: Annotated[Optional[StrictInt],
                      Field(description="The number to skip")] = None,
    limit: Annotated[Optional[StrictInt],
                     Field(description="The number to limit by")] = None,
    name: Annotated[Optional[StrictStr],
                    Field(description="Search by name")] = None,
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

Get workflows

Get all workflows for a tenant

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `offset` (`int`): The number to skip
- `limit` (`int`): The number to limit by
- `name` (`str`): Search by name
- `_request_timeout` (`int, tuple(int, int), optional`): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `str`0 (`str`1): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `str`2 (`str`3): force content-type for the request.
- `str`4 (`str`1): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `str`6 (`str`7): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### workflow\_run\_get

```python
@validate_call
async def workflow_run_get(tenant: Annotated[
    str,
    Field(
        min_length=36, strict=True, max_length=36, description="The tenant id"
    ),
],
                           workflow_run: Annotated[
                               str,
                               Field(
                                   min_length=36,
                                   strict=True,
                                   max_length=36,
                                   description="The workflow run id",
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
                           ) -> WorkflowRun
```

Get workflow run

Get a workflow run for a tenant

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `workflow_run` (`str`): The workflow run id (required)
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

#### workflow\_run\_get\_with\_http\_info

```python
@validate_call
async def workflow_run_get_with_http_info(
    tenant: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The tenant id"),
    ],
    workflow_run: Annotated[
        str,
        Field(
            min_length=36,
            strict=True,
            max_length=36,
            description="The workflow run id",
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
) -> ApiResponse[WorkflowRun]
```

Get workflow run

Get a workflow run for a tenant

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `workflow_run` (`str`): The workflow run id (required)
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

#### workflow\_run\_get\_without\_preload\_content

```python
@validate_call
async def workflow_run_get_without_preload_content(
    tenant: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The tenant id"),
    ],
    workflow_run: Annotated[
        str,
        Field(
            min_length=36,
            strict=True,
            max_length=36,
            description="The workflow run id",
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

Get workflow run

Get a workflow run for a tenant

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `workflow_run` (`str`): The workflow run id (required)
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

#### workflow\_run\_get\_metrics

```python
@validate_call
async def workflow_run_get_metrics(
    tenant: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The tenant id"),
    ],
    event_id: Annotated[
        Optional[Annotated[str,
                           Field(min_length=36, strict=True, max_length=36)]],
        Field(description="The event id to get runs for."),
    ] = None,
    workflow_id: Annotated[
        Optional[Annotated[str,
                           Field(min_length=36, strict=True, max_length=36)]],
        Field(description="The workflow id to get runs for."),
    ] = None,
    parent_workflow_run_id: Annotated[
        Optional[Annotated[str,
                           Field(min_length=36, strict=True, max_length=36)]],
        Field(description="The parent workflow run id"),
    ] = None,
    parent_step_run_id: Annotated[
        Optional[Annotated[str,
                           Field(min_length=36, strict=True, max_length=36)]],
        Field(description="The parent step run id"),
    ] = None,
    additional_metadata: Annotated[
        Optional[List[StrictStr]],
        Field(description="A list of metadata key value pairs to filter by"),
    ] = None,
    created_after: Annotated[
        Optional[datetime],
        Field(description="The time after the workflow run was created"),
    ] = None,
    created_before: Annotated[
        Optional[datetime],
        Field(description="The time before the workflow run was created"),
    ] = None,
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
) -> WorkflowRunsMetrics
```

Get workflow runs metrics

Get a summary of  workflow run metrics for a tenant

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `event_id` (`str`): The event id to get runs for.
- `workflow_id` (`str`): The workflow id to get runs for.
- `parent_workflow_run_id` (`str`): The parent workflow run id
- `parent_step_run_id` (`str`): The parent step run id
- `str`0 (`str`1): A list of metadata key value pairs to filter by
- `str`2 (`str`3): The time after the workflow run was created
- `str`4 (`str`3): The time before the workflow run was created
- `str`6 (`str`7): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `str`8 (`str`9): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `event_id`0 (`event_id`1): force content-type for the request.
- `event_id`2 (`str`9): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `event_id`4 (`event_id`5): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### workflow\_run\_get\_metrics\_with\_http\_info

```python
@validate_call
async def workflow_run_get_metrics_with_http_info(
    tenant: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The tenant id"),
    ],
    event_id: Annotated[
        Optional[Annotated[str,
                           Field(min_length=36, strict=True, max_length=36)]],
        Field(description="The event id to get runs for."),
    ] = None,
    workflow_id: Annotated[
        Optional[Annotated[str,
                           Field(min_length=36, strict=True, max_length=36)]],
        Field(description="The workflow id to get runs for."),
    ] = None,
    parent_workflow_run_id: Annotated[
        Optional[Annotated[str,
                           Field(min_length=36, strict=True, max_length=36)]],
        Field(description="The parent workflow run id"),
    ] = None,
    parent_step_run_id: Annotated[
        Optional[Annotated[str,
                           Field(min_length=36, strict=True, max_length=36)]],
        Field(description="The parent step run id"),
    ] = None,
    additional_metadata: Annotated[
        Optional[List[StrictStr]],
        Field(description="A list of metadata key value pairs to filter by"),
    ] = None,
    created_after: Annotated[
        Optional[datetime],
        Field(description="The time after the workflow run was created"),
    ] = None,
    created_before: Annotated[
        Optional[datetime],
        Field(description="The time before the workflow run was created"),
    ] = None,
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
) -> ApiResponse[WorkflowRunsMetrics]
```

Get workflow runs metrics

Get a summary of  workflow run metrics for a tenant

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `event_id` (`str`): The event id to get runs for.
- `workflow_id` (`str`): The workflow id to get runs for.
- `parent_workflow_run_id` (`str`): The parent workflow run id
- `parent_step_run_id` (`str`): The parent step run id
- `str`0 (`str`1): A list of metadata key value pairs to filter by
- `str`2 (`str`3): The time after the workflow run was created
- `str`4 (`str`3): The time before the workflow run was created
- `str`6 (`str`7): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `str`8 (`str`9): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `event_id`0 (`event_id`1): force content-type for the request.
- `event_id`2 (`str`9): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `event_id`4 (`event_id`5): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### workflow\_run\_get\_metrics\_without\_preload\_content

```python
@validate_call
async def workflow_run_get_metrics_without_preload_content(
    tenant: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The tenant id"),
    ],
    event_id: Annotated[
        Optional[Annotated[str,
                           Field(min_length=36, strict=True, max_length=36)]],
        Field(description="The event id to get runs for."),
    ] = None,
    workflow_id: Annotated[
        Optional[Annotated[str,
                           Field(min_length=36, strict=True, max_length=36)]],
        Field(description="The workflow id to get runs for."),
    ] = None,
    parent_workflow_run_id: Annotated[
        Optional[Annotated[str,
                           Field(min_length=36, strict=True, max_length=36)]],
        Field(description="The parent workflow run id"),
    ] = None,
    parent_step_run_id: Annotated[
        Optional[Annotated[str,
                           Field(min_length=36, strict=True, max_length=36)]],
        Field(description="The parent step run id"),
    ] = None,
    additional_metadata: Annotated[
        Optional[List[StrictStr]],
        Field(description="A list of metadata key value pairs to filter by"),
    ] = None,
    created_after: Annotated[
        Optional[datetime],
        Field(description="The time after the workflow run was created"),
    ] = None,
    created_before: Annotated[
        Optional[datetime],
        Field(description="The time before the workflow run was created"),
    ] = None,
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

Get workflow runs metrics

Get a summary of  workflow run metrics for a tenant

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `event_id` (`str`): The event id to get runs for.
- `workflow_id` (`str`): The workflow id to get runs for.
- `parent_workflow_run_id` (`str`): The parent workflow run id
- `parent_step_run_id` (`str`): The parent step run id
- `str`0 (`str`1): A list of metadata key value pairs to filter by
- `str`2 (`str`3): The time after the workflow run was created
- `str`4 (`str`3): The time before the workflow run was created
- `str`6 (`str`7): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `str`8 (`str`9): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `event_id`0 (`event_id`1): force content-type for the request.
- `event_id`2 (`str`9): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `event_id`4 (`event_id`5): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### workflow\_run\_get\_shape

```python
@validate_call
async def workflow_run_get_shape(tenant: Annotated[
    str,
    Field(
        min_length=36, strict=True, max_length=36, description="The tenant id"
    ),
],
                                 workflow_run: Annotated[
                                     str,
                                     Field(
                                         min_length=36,
                                         strict=True,
                                         max_length=36,
                                         description="The workflow run id",
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
                                 ) -> WorkflowRunShape
```

Get workflow run

Get a workflow run for a tenant

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `workflow_run` (`str`): The workflow run id (required)
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

#### workflow\_run\_get\_shape\_with\_http\_info

```python
@validate_call
async def workflow_run_get_shape_with_http_info(
    tenant: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The tenant id"),
    ],
    workflow_run: Annotated[
        str,
        Field(
            min_length=36,
            strict=True,
            max_length=36,
            description="The workflow run id",
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
) -> ApiResponse[WorkflowRunShape]
```

Get workflow run

Get a workflow run for a tenant

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `workflow_run` (`str`): The workflow run id (required)
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

#### workflow\_run\_get\_shape\_without\_preload\_content

```python
@validate_call
async def workflow_run_get_shape_without_preload_content(
    tenant: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The tenant id"),
    ],
    workflow_run: Annotated[
        str,
        Field(
            min_length=36,
            strict=True,
            max_length=36,
            description="The workflow run id",
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

Get workflow run

Get a workflow run for a tenant

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `workflow_run` (`str`): The workflow run id (required)
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

#### workflow\_run\_list

```python
@validate_call
async def workflow_run_list(
    tenant: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The tenant id"),
    ],
    offset: Annotated[Optional[StrictInt],
                      Field(description="The number to skip")] = None,
    limit: Annotated[Optional[StrictInt],
                     Field(description="The number to limit by")] = None,
    event_id: Annotated[
        Optional[Annotated[str,
                           Field(min_length=36, strict=True, max_length=36)]],
        Field(description="The event id to get runs for."),
    ] = None,
    workflow_id: Annotated[
        Optional[Annotated[str,
                           Field(min_length=36, strict=True, max_length=36)]],
        Field(description="The workflow id to get runs for."),
    ] = None,
    parent_workflow_run_id: Annotated[
        Optional[Annotated[str,
                           Field(min_length=36, strict=True, max_length=36)]],
        Field(description="The parent workflow run id"),
    ] = None,
    parent_step_run_id: Annotated[
        Optional[Annotated[str,
                           Field(min_length=36, strict=True, max_length=36)]],
        Field(description="The parent step run id"),
    ] = None,
    statuses: Annotated[
        Optional[List[WorkflowRunStatus]],
        Field(description="A list of workflow run statuses to filter by"),
    ] = None,
    kinds: Annotated[
        Optional[List[WorkflowKind]],
        Field(description="A list of workflow kinds to filter by"),
    ] = None,
    additional_metadata: Annotated[
        Optional[List[StrictStr]],
        Field(description="A list of metadata key value pairs to filter by"),
    ] = None,
    created_after: Annotated[
        Optional[datetime],
        Field(description="The time after the workflow run was created"),
    ] = None,
    created_before: Annotated[
        Optional[datetime],
        Field(description="The time before the workflow run was created"),
    ] = None,
    finished_after: Annotated[
        Optional[datetime],
        Field(description="The time after the workflow run was finished"),
    ] = None,
    finished_before: Annotated[
        Optional[datetime],
        Field(description="The time before the workflow run was finished"),
    ] = None,
    order_by_field: Annotated[Optional[WorkflowRunOrderByField],
                              Field(description="The order by field")] = None,
    order_by_direction: Annotated[
        Optional[WorkflowRunOrderByDirection],
        Field(description="The order by direction"),
    ] = None,
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
                           Field(ge=0, le=0)] = 0) -> WorkflowRunList
```

Get workflow runs

Get all workflow runs for a tenant

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `offset` (`int`): The number to skip
- `limit` (`int`): The number to limit by
- `event_id` (`str`): The event id to get runs for.
- `workflow_id` (`str`): The workflow id to get runs for.
- `str`0 (`str`): The parent workflow run id
- `str`2 (`str`): The parent step run id
- `str`4 (`str`5): A list of workflow run statuses to filter by
- `str`6 (`str`7): A list of workflow kinds to filter by
- `str`8 (`str`9): A list of metadata key value pairs to filter by
- `offset`0 (`offset`1): The time after the workflow run was created
- `offset`2 (`offset`1): The time before the workflow run was created
- `offset`4 (`offset`1): The time after the workflow run was finished
- `offset`6 (`offset`1): The time before the workflow run was finished
- `offset`8 (`offset`9): The order by field
- `int`0 (`int`1): The order by direction
- `int`2 (`int`3): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `int`4 (`int`5): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `int`6 (`int`7): force content-type for the request.
- `int`8 (`int`5): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `limit`0 (`limit`1): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### workflow\_run\_list\_with\_http\_info

```python
@validate_call
async def workflow_run_list_with_http_info(
    tenant: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The tenant id"),
    ],
    offset: Annotated[Optional[StrictInt],
                      Field(description="The number to skip")] = None,
    limit: Annotated[Optional[StrictInt],
                     Field(description="The number to limit by")] = None,
    event_id: Annotated[
        Optional[Annotated[str,
                           Field(min_length=36, strict=True, max_length=36)]],
        Field(description="The event id to get runs for."),
    ] = None,
    workflow_id: Annotated[
        Optional[Annotated[str,
                           Field(min_length=36, strict=True, max_length=36)]],
        Field(description="The workflow id to get runs for."),
    ] = None,
    parent_workflow_run_id: Annotated[
        Optional[Annotated[str,
                           Field(min_length=36, strict=True, max_length=36)]],
        Field(description="The parent workflow run id"),
    ] = None,
    parent_step_run_id: Annotated[
        Optional[Annotated[str,
                           Field(min_length=36, strict=True, max_length=36)]],
        Field(description="The parent step run id"),
    ] = None,
    statuses: Annotated[
        Optional[List[WorkflowRunStatus]],
        Field(description="A list of workflow run statuses to filter by"),
    ] = None,
    kinds: Annotated[
        Optional[List[WorkflowKind]],
        Field(description="A list of workflow kinds to filter by"),
    ] = None,
    additional_metadata: Annotated[
        Optional[List[StrictStr]],
        Field(description="A list of metadata key value pairs to filter by"),
    ] = None,
    created_after: Annotated[
        Optional[datetime],
        Field(description="The time after the workflow run was created"),
    ] = None,
    created_before: Annotated[
        Optional[datetime],
        Field(description="The time before the workflow run was created"),
    ] = None,
    finished_after: Annotated[
        Optional[datetime],
        Field(description="The time after the workflow run was finished"),
    ] = None,
    finished_before: Annotated[
        Optional[datetime],
        Field(description="The time before the workflow run was finished"),
    ] = None,
    order_by_field: Annotated[Optional[WorkflowRunOrderByField],
                              Field(description="The order by field")] = None,
    order_by_direction: Annotated[
        Optional[WorkflowRunOrderByDirection],
        Field(description="The order by direction"),
    ] = None,
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
) -> ApiResponse[WorkflowRunList]
```

Get workflow runs

Get all workflow runs for a tenant

**Arguments**:

- `tenant` (`str`): The tenant id (required)
- `offset` (`int`): The number to skip
- `limit` (`int`): The number to limit by
- `event_id` (`str`): The event id to get runs for.
- `workflow_id` (`str`): The workflow id to get runs for.
- `str`0 (`str`): The parent workflow run id
- `str`2 (`str`): The parent step run id
- `str`4 (`str`5): A list of workflow run statuses to filter by
- `str`6 (`str`7): A list of workflow kinds to filter by
- `str`8 (`str`9): A list of metadata key value pairs to filter by
- `offset`0 (`offset`1): The time after the workflow run was created
- `offset`2 (`offset`1): The time before the workflow run was created
- `offset`4 (`offset`1): The time after the workflow run was finished
- `offset`6 (`offset`1): The time before the workflow run was finished
- `offset`8 (`offset`9): The order by field
- `int`0 (`int`1): The order by direction
- `int`2 (`int`3): timeout setting for this request. If one
number provided, it will be total request
timeout. It can also be a pair (tuple) of
(connection, read) timeouts.
- `int`4 (`int`5): set to override the auth_settings for an a single
request; this effectively ignores the
authentication in the spec for a single request.
- `int`6 (`int`7): force content-type for the request.
- `int`8 (`int`5): set to override the headers for a single
request; this effectively ignores the headers
in the spec for a single request.
- `limit`0 (`limit`1): set to override the host_index for a single
request; this effectively ignores the host_index
in the spec for a single request.

**Returns**:

Returns the result object.

#### workflow\_run\_list\_without\_preload\_content

```python
@validate_call
async def workflow_run_list_without_preload_content(
    tenant: Annotated[
        str,
        Field(min_length=36,
              strict=True,
              max_length=36,
              description="The tenant id"),
    ],
    offset: Annotated[Optional[StrictInt],
                      Field(description="The number to skip")] = None,
    limit: Annotated[Optional[StrictInt],
                     Field(description="The number to limit by")] = None,
    event_id: Annotated[
        Optional[Annotated[str,
                           Field(min_length=36, strict=True, max_length=36)]],
        Field(description="The event id to get runs for."),
    ] = None,
    workflow_id: Annotated[
        Optional[Annotated[str,
                           Field(min_length=36, strict=True, max_length=36)]],
        Field(description="The workflow id to get runs for."),
    ] = None,
    parent_workflow_run_id: Annotated[
        Optional[Annotated[str,
                           Field(min_length=36, strict=True, max_length=36)]],
        Field(description="The parent workflow run id"),
    ] = None,
    parent_step_run_id: Annotated[
        Optional[Annotated[str,
                           Field(min_length=36, strict=True, max_length=36)]],
        Field(description="The parent step run id"),
    ] = None,
    statuses: Annotated[
        Optional[List[WorkflowRunStatus]],
        Field(description="A list of workflow run statuses to filter by"),
    ] = None,
    kinds: Annotated[
        Optional[List[WorkflowKind]],
        Field(description="A list of workflow kinds to filter by"),
    ] = None,
    additional_metadata: Annotated[
        Optional[List[StrictStr]],
        Field(description="A list of metadata key value pairs to filter by"),
    ] = None,
    created_after: Annotated[
        Optional[datetime],
        Field(description="The time after the workflow run was created"),
    ] = None,
    created_before: Annotated[
        Optional[datetime],
        Field(description="The time before the workflow run was created"),
    ] = None,
    finished_after: Annotated[
        Optional[datetime],
        Field(description="The time after the workflow run was finished"),
    ] = None,
    finished_before: Annotated[
        Optional[datetime],
        Field(description="The time before the workflow run was finished"),
    ] = None,
    order_by_field: Annotated[Optional[WorkflowRunOrderByField],
                              Field(description="The order by field")] = None,
    order_by_direction: Annotated[
        Optional[WorkflowRunOrderByDirection],
        Field(description="The order by direction"),
    ] = None,
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