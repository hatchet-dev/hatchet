[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / Api

# Class: Api\<SecurityDataType\>

Defined in: [src/clients/rest/generated/Api.ts:126](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L126)

## Extends

- `HttpClient`\<`SecurityDataType`\>

## Type Parameters

### SecurityDataType

`SecurityDataType` = `unknown`

## Constructors

### Constructor

> **new Api**\<`SecurityDataType`\>(`__namedParameters`): `Api`\<`SecurityDataType`\>

Defined in: [src/clients/rest/generated/http-client.ts:65](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/http-client.ts#L65)

#### Parameters

##### \_\_namedParameters

`ApiConfig`\<`SecurityDataType`\> = `{}`

#### Returns

`Api`\<`SecurityDataType`\>

#### Inherited from

`HttpClient<SecurityDataType>.constructor`

## Properties

### instance

> **instance**: `AxiosInstance`

Defined in: [src/clients/rest/generated/http-client.ts:59](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/http-client.ts#L59)

#### Inherited from

`HttpClient.instance`

## Methods

### alertEmailGroupCreate()

> **alertEmailGroupCreate**(`tenant`, `data`, `params`): `Promise`\<`AxiosResponse`\<[`TenantAlertEmailGroup`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/TenantAlertEmailGroup.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:735](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L735)

#### Parameters

##### tenant

`string`

##### data

[`CreateTenantAlertEmailGroupRequest`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/CreateTenantAlertEmailGroupRequest.md)

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`TenantAlertEmailGroup`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/TenantAlertEmailGroup.md), `any`\>\>

#### Description

Creates a new tenant alert email group

#### Tags

Tenant

#### Name

AlertEmailGroupCreate

#### Request

POST:/api/v1/tenants/{tenant}/alerting-email-groups

#### Secure

***

### alertEmailGroupDelete()

> **alertEmailGroupDelete**(`alertEmailGroup`, `params`): `Promise`\<`AxiosResponse`\<`void`, `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:815](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L815)

#### Parameters

##### alertEmailGroup

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<`void`, `any`\>\>

#### Description

Deletes a tenant alert email group

#### Tags

Tenant

#### Name

AlertEmailGroupDelete

#### Request

DELETE:/api/v1/alerting-email-groups/{alert-email-group}

#### Secure

***

### alertEmailGroupList()

> **alertEmailGroupList**(`tenant`, `params`): `Promise`\<`AxiosResponse`\<[`TenantAlertEmailGroupList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/TenantAlertEmailGroupList.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:758](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L758)

#### Parameters

##### tenant

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`TenantAlertEmailGroupList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/TenantAlertEmailGroupList.md), `any`\>\>

#### Description

Gets a list of tenant alert email groups

#### Tags

Tenant

#### Name

AlertEmailGroupList

#### Request

GET:/api/v1/tenants/{tenant}/alerting-email-groups

#### Secure

***

### alertEmailGroupUpdate()

> **alertEmailGroupUpdate**(`alertEmailGroup`, `data`, `params`): `Promise`\<`AxiosResponse`\<[`TenantAlertEmailGroup`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/TenantAlertEmailGroup.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:792](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L792)

#### Parameters

##### alertEmailGroup

`string`

##### data

[`UpdateTenantAlertEmailGroupRequest`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/UpdateTenantAlertEmailGroupRequest.md)

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`TenantAlertEmailGroup`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/TenantAlertEmailGroup.md), `any`\>\>

#### Description

Updates a tenant alert email group

#### Tags

Tenant

#### Name

AlertEmailGroupUpdate

#### Request

PATCH:/api/v1/alerting-email-groups/{alert-email-group}

#### Secure

***

### apiTokenCreate()

> **apiTokenCreate**(`tenant`, `data`, `params`): `Promise`\<`AxiosResponse`\<[`CreateAPITokenResponse`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/CreateAPITokenResponse.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1154](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1154)

#### Parameters

##### tenant

`string`

##### data

[`CreateAPITokenRequest`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/CreateAPITokenRequest.md)

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`CreateAPITokenResponse`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/CreateAPITokenResponse.md), `any`\>\>

#### Description

Create an API token for a tenant

#### Tags

API Token

#### Name

ApiTokenCreate

#### Request

POST:/api/v1/tenants/{tenant}/api-tokens

#### Secure

***

### apiTokenList()

> **apiTokenList**(`tenant`, `params`): `Promise`\<`AxiosResponse`\<[`ListAPITokensResponse`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/ListAPITokensResponse.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1173](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1173)

#### Parameters

##### tenant

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`ListAPITokensResponse`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/ListAPITokensResponse.md), `any`\>\>

#### Description

List API tokens for a tenant

#### Tags

API Token

#### Name

ApiTokenList

#### Request

GET:/api/v1/tenants/{tenant}/api-tokens

#### Secure

***

### apiTokenUpdateRevoke()

> **apiTokenUpdateRevoke**(`apiToken`, `params`): `Promise`\<`AxiosResponse`\<`void`, `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1190](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1190)

#### Parameters

##### apiToken

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<`void`, `any`\>\>

#### Description

Revoke an API token for a tenant

#### Tags

API Token

#### Name

ApiTokenUpdateRevoke

#### Request

POST:/api/v1/api-tokens/{api-token}

#### Secure

***

### cloudMetadataGet()

> **cloudMetadataGet**(`params`): `Promise`\<`AxiosResponse`\<[`APIErrors`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/APIErrors.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:547](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L547)

#### Parameters

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`APIErrors`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/APIErrors.md), `any`\>\>

#### Description

Gets metadata for the Hatchet cloud instance

#### Tags

Metadata

#### Name

CloudMetadataGet

#### Request

GET:/api/v1/cloud/metadata

***

### cronWorkflowList()

> **cronWorkflowList**(`tenant`, `query?`, `params?`): `Promise`\<`AxiosResponse`\<[`CronWorkflowsList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/CronWorkflowsList.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1701](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1701)

#### Parameters

##### tenant

`string`

##### query?

###### additionalMetadata?

`string`[]

A list of metadata key value pairs to filter by

**Example**

```ts
["key1:value1","key2:value2"]
```

###### limit?

`number`

The number to limit by

**Format**

int64

###### offset?

`number`

The number to skip

**Format**

int64

###### orderByDirection?

[`WorkflowRunOrderByDirection`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/enumerations/WorkflowRunOrderByDirection.md)

The order by direction

###### orderByField?

[`CronWorkflowsOrderByField`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/enumerations/CronWorkflowsOrderByField.md)

The order by field

###### workflowId?

`string`

The workflow id to get runs for.

**Format**

uuid

**Min Length**

36

**Max Length**

36

##### params?

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`CronWorkflowsList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/CronWorkflowsList.md), `any`\>\>

#### Description

Get all cron job workflow triggers for a tenant

#### Tags

Workflow

#### Name

CronWorkflowList

#### Request

GET:/api/v1/tenants/{tenant}/workflows/crons

#### Secure

***

### cronWorkflowTriggerCreate()

> **cronWorkflowTriggerCreate**(`tenant`, `workflow`, `data`, `params`): `Promise`\<`AxiosResponse`\<[`CronWorkflows`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/CronWorkflows.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1677](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1677)

#### Parameters

##### tenant

`string`

##### workflow

`string`

##### data

[`CreateCronWorkflowTriggerRequest`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/CreateCronWorkflowTriggerRequest.md)

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`CronWorkflows`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/CronWorkflows.md), `any`\>\>

#### Description

Create a new cron job workflow trigger for a tenant

#### Tags

Workflow Run

#### Name

CronWorkflowTriggerCreate

#### Request

POST:/api/v1/tenants/{tenant}/workflows/{workflow}/crons

#### Secure

***

### eventCreate()

> **eventCreate**(`tenant`, `data`, `params`): `Promise`\<`AxiosResponse`\<[`Event`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/Event.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1305](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1305)

#### Parameters

##### tenant

`string`

##### data

[`CreateEventRequest`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/CreateEventRequest.md)

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`Event`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/Event.md), `any`\>\>

#### Description

Creates a new event.

#### Tags

Event

#### Name

EventCreate

#### Request

POST:/api/v1/tenants/{tenant}/events

#### Secure

***

### eventCreateBulk()

> **eventCreateBulk**(`tenant`, `data`, `params`): `Promise`\<`AxiosResponse`\<[`BulkCreateEventResponse`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/BulkCreateEventResponse.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1324](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1324)

#### Parameters

##### tenant

`string`

##### data

[`BulkCreateEventRequest`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/BulkCreateEventRequest.md)

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`BulkCreateEventResponse`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/BulkCreateEventResponse.md), `any`\>\>

#### Description

Bulk creates new events.

#### Tags

Event

#### Name

EventCreateBulk

#### Request

POST:/api/v1/tenants/{tenant}/events/bulk

#### Secure

***

### eventDataGet()

> **eventDataGet**(`event`, `params`): `Promise`\<`AxiosResponse`\<[`EventData`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/EventData.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1476](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1476)

#### Parameters

##### event

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`EventData`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/EventData.md), `any`\>\>

#### Description

Get the data for an event.

#### Tags

Event

#### Name

EventDataGet

#### Request

GET:/api/v1/events/{event}/data

#### Secure

***

### eventGet()

> **eventGet**(`event`, `params`): `Promise`\<`AxiosResponse`\<[`Event`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/Event.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1459](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1459)

#### Parameters

##### event

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`Event`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/Event.md), `any`\>\>

#### Description

Get an event.

#### Tags

Event

#### Name

EventGet

#### Request

GET:/api/v1/events/{event}

#### Secure

***

### eventKeyList()

> **eventKeyList**(`tenant`, `params`): `Promise`\<`AxiosResponse`\<[`EventKeyList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/EventKeyList.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1493](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1493)

#### Parameters

##### tenant

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`EventKeyList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/EventKeyList.md), `any`\>\>

#### Description

Lists all event keys for a tenant.

#### Tags

Event

#### Name

EventKeyList

#### Request

GET:/api/v1/tenants/{tenant}/events/keys

#### Secure

***

### eventList()

> **eventList**(`tenant`, `query?`, `params?`): `Promise`\<`AxiosResponse`\<[`EventList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/EventList.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1253](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1253)

#### Parameters

##### tenant

`string`

##### query?

###### additionalMetadata?

`string`[]

A list of metadata key value pairs to filter by

**Example**

```ts
["key1:value1","key2:value2"]
```

###### eventIds?

`string`[]

A list of event ids to filter by

###### keys?

`string`[]

A list of keys to filter by

###### limit?

`number`

The number to limit by

**Format**

int64

###### offset?

`number`

The number to skip

**Format**

int64

###### orderByDirection?

[`EventOrderByDirection`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/enumerations/EventOrderByDirection.md)

The order direction

###### orderByField?

[`CreatedAt`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/enumerations/EventOrderByField.md#createdat)

What to order by

###### search?

`string`

The search query to filter for

###### statuses?

[`WorkflowRunStatusList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/type-aliases/WorkflowRunStatusList.md)

A list of workflow run statuses to filter by

###### workflows?

`string`[]

A list of workflow IDs to filter by

##### params?

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`EventList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/EventList.md), `any`\>\>

#### Description

Lists all events for a tenant.

#### Tags

Event

#### Name

EventList

#### Request

GET:/api/v1/tenants/{tenant}/events

#### Secure

***

### eventUpdateCancel()

> **eventUpdateCancel**(`tenant`, `data`, `params`): `Promise`\<`AxiosResponse`\<\{ `workflowRunIds`: `string`[]; \}, `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1362](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1362)

#### Parameters

##### tenant

`string`

##### data

[`CancelEventRequest`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/CancelEventRequest.md)

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<\{ `workflowRunIds`: `string`[]; \}, `any`\>\>

#### Description

Cancels all runs for a list of events.

#### Tags

Event

#### Name

EventUpdateCancel

#### Request

POST:/api/v1/tenants/{tenant}/events/cancel

#### Secure

***

### eventUpdateReplay()

> **eventUpdateReplay**(`tenant`, `data`, `params`): `Promise`\<`AxiosResponse`\<[`EventList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/EventList.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1343](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1343)

#### Parameters

##### tenant

`string`

##### data

[`ReplayEventRequest`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/ReplayEventRequest.md)

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`EventList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/EventList.md), `any`\>\>

#### Description

Replays a list of events.

#### Tags

Event

#### Name

EventUpdateReplay

#### Request

POST:/api/v1/tenants/{tenant}/events/replay

#### Secure

***

### infoGetVersion()

> **infoGetVersion**(`params`): `Promise`\<`AxiosResponse`\<\{ `version`: `string`; \}, `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:2552](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L2552)

#### Parameters

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<\{ `version`: `string`; \}, `any`\>\>

#### Description

Get the version of the server

#### Name

InfoGetVersion

#### Request

GET:/api/v1/version

***

### livenessGet()

> **livenessGet**(`params`): `Promise`\<`AxiosResponse`\<`void`, `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:518](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L518)

#### Parameters

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<`void`, `any`\>\>

#### Description

Gets the liveness status

#### Tags

Healthcheck

#### Name

LivenessGet

#### Request

GET:/api/live

***

### logLineList()

> **logLineList**(`stepRun`, `query?`, `params?`): `Promise`\<`AxiosResponse`\<[`LogLineList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/LogLineList.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1953](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1953)

#### Parameters

##### stepRun

`string`

##### query?

###### levels?

[`LogLineLevelField`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/type-aliases/LogLineLevelField.md)

A list of levels to filter by

###### limit?

`number`

The number to limit by

**Format**

int64

###### offset?

`number`

The number to skip

**Format**

int64

###### orderByDirection?

[`LogLineOrderByDirection`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/enumerations/LogLineOrderByDirection.md)

The order direction

###### orderByField?

[`CreatedAt`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/enumerations/LogLineOrderByField.md#createdat)

What to order by

###### search?

`string`

The search query to filter for

##### params?

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`LogLineList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/LogLineList.md), `any`\>\>

#### Description

Lists log lines for a step run.

#### Tags

Log

#### Name

LogLineList

#### Request

GET:/api/v1/step-runs/{step-run}/logs

#### Secure

***

### metadataGet()

> **metadataGet**(`params`): `Promise`\<`AxiosResponse`\<[`APIMeta`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/APIMeta.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:532](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L532)

#### Parameters

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`APIMeta`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/APIMeta.md), `any`\>\>

#### Description

Gets metadata for the Hatchet instance

#### Tags

Metadata

#### Name

MetadataGet

#### Request

GET:/api/v1/meta

***

### metadataListIntegrations()

> **metadataListIntegrations**(`params`): `Promise`\<`AxiosResponse`\<[`ListAPIMetaIntegration`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/type-aliases/ListAPIMetaIntegration.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:563](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L563)

#### Parameters

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`ListAPIMetaIntegration`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/type-aliases/ListAPIMetaIntegration.md), `any`\>\>

#### Description

List all integrations

#### Tags

Metadata

#### Name

MetadataListIntegrations

#### Request

GET:/api/v1/meta/integrations

#### Secure

***

### monitoringPostRunProbe()

> **monitoringPostRunProbe**(`tenant`, `params`): `Promise`\<`AxiosResponse`\<`void`, `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:2538](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L2538)

#### Parameters

##### tenant

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<`void`, `any`\>\>

#### Description

Triggers a workflow to check the status of the instance

#### Name

MonitoringPostRunProbe

#### Request

POST:/api/v1/monitoring/{tenant}/probe

#### Secure

***

### rateLimitList()

> **rateLimitList**(`tenant`, `query?`, `params?`): `Promise`\<`AxiosResponse`\<[`RateLimitList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/RateLimitList.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1386](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1386)

#### Parameters

##### tenant

`string`

##### query?

###### limit?

`number`

The number to limit by

**Format**

int64

###### offset?

`number`

The number to skip

**Format**

int64

###### orderByDirection?

[`RateLimitOrderByDirection`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/enumerations/RateLimitOrderByDirection.md)

The order direction

###### orderByField?

[`RateLimitOrderByField`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/enumerations/RateLimitOrderByField.md)

What to order by

###### search?

`string`

The search query to filter for

##### params?

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`RateLimitList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/RateLimitList.md), `any`\>\>

#### Description

Lists all rate limits for a tenant.

#### Tags

Rate Limits

#### Name

RateLimitList

#### Request

GET:/api/v1/tenants/{tenant}/rate-limits

#### Secure

***

### readinessGet()

> **readinessGet**(`params`): `Promise`\<`AxiosResponse`\<`void`, `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:504](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L504)

#### Parameters

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<`void`, `any`\>\>

#### Description

Gets the readiness status

#### Tags

Healthcheck

#### Name

ReadinessGet

#### Request

GET:/api/ready

***

### request()

> **request**\<`T`, `_E`\>(`__namedParameters`): `Promise`\<`AxiosResponse`\<`T`, `any`\>\>

Defined in: [src/clients/rest/generated/http-client.ts:129](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/http-client.ts#L129)

#### Type Parameters

##### T

`T` = `any`

##### _E

`_E` = `any`

#### Parameters

##### \_\_namedParameters

`FullRequestParams`

#### Returns

`Promise`\<`AxiosResponse`\<`T`, `any`\>\>

#### Inherited from

`HttpClient.request`

***

### scheduledWorkflowRunCreate()

> **scheduledWorkflowRunCreate**(`tenant`, `workflow`, `data`, `params`): `Promise`\<`AxiosResponse`\<[`ScheduledWorkflows`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/ScheduledWorkflows.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1547](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1547)

#### Parameters

##### tenant

`string`

##### workflow

`string`

##### data

[`ScheduleWorkflowRunRequest`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/ScheduleWorkflowRunRequest.md)

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`ScheduledWorkflows`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/ScheduledWorkflows.md), `any`\>\>

#### Description

Schedule a new workflow run for a tenant

#### Tags

Workflow Run

#### Name

ScheduledWorkflowRunCreate

#### Request

POST:/api/v1/tenants/{tenant}/workflows/{workflow}/scheduled

#### Secure

***

### setSecurityData()

> **setSecurityData**(`data`): `void`

Defined in: [src/clients/rest/generated/http-client.ts:80](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/http-client.ts#L80)

#### Parameters

##### data

`null` | `SecurityDataType`

#### Returns

`void`

#### Inherited from

`HttpClient.setSecurityData`

***

### slackWebhookDelete()

> **slackWebhookDelete**(`slack`, `params`): `Promise`\<`AxiosResponse`\<`void`, `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:864](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L864)

#### Parameters

##### slack

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<`void`, `any`\>\>

#### Description

Delete Slack webhook

#### Tags

Slack

#### Name

SlackWebhookDelete

#### Request

DELETE:/api/v1/slack/{slack}

#### Secure

***

### slackWebhookList()

> **slackWebhookList**(`tenant`, `params`): `Promise`\<`AxiosResponse`\<[`ListSlackWebhooks`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/ListSlackWebhooks.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:847](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L847)

#### Parameters

##### tenant

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`ListSlackWebhooks`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/ListSlackWebhooks.md), `any`\>\>

#### Description

List Slack webhooks

#### Tags

Slack

#### Name

SlackWebhookList

#### Request

GET:/api/v1/tenants/{tenant}/slack

#### Secure

***

### snsCreate()

> **snsCreate**(`tenant`, `data`, `params`): `Promise`\<`AxiosResponse`\<[`SNSIntegration`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/SNSIntegration.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:716](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L716)

#### Parameters

##### tenant

`string`

##### data

[`CreateSNSIntegrationRequest`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/CreateSNSIntegrationRequest.md)

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`SNSIntegration`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/SNSIntegration.md), `any`\>\>

#### Description

Create SNS integration

#### Tags

SNS

#### Name

SnsCreate

#### Request

POST:/api/v1/tenants/{tenant}/sns

#### Secure

***

### snsDelete()

> **snsDelete**(`sns`, `params`): `Promise`\<`AxiosResponse`\<`void`, `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:831](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L831)

#### Parameters

##### sns

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<`void`, `any`\>\>

#### Description

Delete SNS integration

#### Tags

SNS

#### Name

SnsDelete

#### Request

DELETE:/api/v1/sns/{sns}

#### Secure

***

### snsList()

> **snsList**(`tenant`, `params`): `Promise`\<`AxiosResponse`\<[`ListSNSIntegrations`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/ListSNSIntegrations.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:699](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L699)

#### Parameters

##### tenant

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`ListSNSIntegrations`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/ListSNSIntegrations.md), `any`\>\>

#### Description

List SNS integrations

#### Tags

SNS

#### Name

SnsList

#### Request

GET:/api/v1/tenants/{tenant}/sns

#### Secure

***

### snsUpdate()

> **snsUpdate**(`tenant`, `event`, `params`): `Promise`\<`AxiosResponse`\<`void`, `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:684](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L684)

#### Parameters

##### tenant

`string`

##### event

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<`void`, `any`\>\>

#### Description

SNS event

#### Tags

Github

#### Name

SnsUpdate

#### Request

POST:/api/v1/sns/{tenant}/{event}

***

### stepRunGet()

> **stepRunGet**(`tenant`, `stepRun`, `params`): `Promise`\<`AxiosResponse`\<[`StepRun`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/StepRun.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:2329](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L2329)

#### Parameters

##### tenant

`string`

##### stepRun

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`StepRun`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/StepRun.md), `any`\>\>

#### Description

Get a step run by id

#### Tags

Step Run

#### Name

StepRunGet

#### Request

GET:/api/v1/tenants/{tenant}/step-runs/{step-run}

#### Secure

***

### stepRunGetSchema()

> **stepRunGetSchema**(`tenant`, `stepRun`, `params`): `Promise`\<`AxiosResponse`\<`object`, `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:2387](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L2387)

#### Parameters

##### tenant

`string`

##### stepRun

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<`object`, `any`\>\>

#### Description

Get the schema for a step run

#### Tags

Step Run

#### Name

StepRunGetSchema

#### Request

GET:/api/v1/tenants/{tenant}/step-runs/{step-run}/schema

#### Secure

***

### stepRunListArchives()

> **stepRunListArchives**(`stepRun`, `query?`, `params?`): `Promise`\<`AxiosResponse`\<[`StepRunArchiveList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/StepRunArchiveList.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:2056](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L2056)

#### Parameters

##### stepRun

`string`

##### query?

###### limit?

`number`

The number to limit by

**Format**

int64

###### offset?

`number`

The number to skip

**Format**

int64

##### params?

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`StepRunArchiveList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/StepRunArchiveList.md), `any`\>\>

#### Description

List archives for a step run

#### Tags

Step Run

#### Name

StepRunListArchives

#### Request

GET:/api/v1/step-runs/{step-run}/archives

#### Secure

***

### stepRunListEvents()

> **stepRunListEvents**(`stepRun`, `query?`, `params?`): `Promise`\<`AxiosResponse`\<[`StepRunEventList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/StepRunEventList.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1994](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1994)

#### Parameters

##### stepRun

`string`

##### query?

###### limit?

`number`

The number to limit by

**Format**

int64

###### offset?

`number`

The number to skip

**Format**

int64

##### params?

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`StepRunEventList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/StepRunEventList.md), `any`\>\>

#### Description

List events for a step run

#### Tags

Step Run

#### Name

StepRunListEvents

#### Request

GET:/api/v1/step-runs/{step-run}/events

#### Secure

***

### stepRunUpdateCancel()

> **stepRunUpdateCancel**(`tenant`, `stepRun`, `params`): `Promise`\<`AxiosResponse`\<[`StepRun`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/StepRun.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:2370](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L2370)

#### Parameters

##### tenant

`string`

##### stepRun

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`StepRun`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/StepRun.md), `any`\>\>

#### Description

Attempts to cancel a step run

#### Tags

Step Run

#### Name

StepRunUpdateCancel

#### Request

POST:/api/v1/tenants/{tenant}/step-runs/{step-run}/cancel

#### Secure

***

### stepRunUpdateRerun()

> **stepRunUpdateRerun**(`tenant`, `stepRun`, `data`, `params`): `Promise`\<`AxiosResponse`\<[`StepRun`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/StepRun.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:2346](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L2346)

#### Parameters

##### tenant

`string`

##### stepRun

`string`

##### data

[`RerunStepRunRequest`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/RerunStepRunRequest.md)

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`StepRun`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/StepRun.md), `any`\>\>

#### Description

Reruns a step run

#### Tags

Step Run

#### Name

StepRunUpdateRerun

#### Request

POST:/api/v1/tenants/{tenant}/step-runs/{step-run}/rerun

#### Secure

***

### tenantAlertingSettingsGet()

> **tenantAlertingSettingsGet**(`tenant`, `params`): `Promise`\<`AxiosResponse`\<[`TenantAlertingSettings`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/TenantAlertingSettings.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1058](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1058)

#### Parameters

##### tenant

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`TenantAlertingSettings`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/TenantAlertingSettings.md), `any`\>\>

#### Description

Gets the alerting settings for a tenant

#### Tags

Tenant

#### Name

TenantAlertingSettingsGet

#### Request

GET:/api/v1/tenants/{tenant}/alerting/settings

#### Secure

***

### tenantCreate()

> **tenantCreate**(`data`, `params`): `Promise`\<`AxiosResponse`\<[`Tenant`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/Tenant.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1020](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1020)

#### Parameters

##### data

[`CreateTenantRequest`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/CreateTenantRequest.md)

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`Tenant`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/Tenant.md), `any`\>\>

#### Description

Creates a new tenant

#### Tags

Tenant

#### Name

TenantCreate

#### Request

POST:/api/v1/tenants

#### Secure

***

### tenantGetQueueMetrics()

> **tenantGetQueueMetrics**(`tenant`, `query?`, `params?`): `Promise`\<`AxiosResponse`\<[`TenantQueueMetrics`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/TenantQueueMetrics.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1206](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1206)

#### Parameters

##### tenant

`string`

##### query?

###### additionalMetadata?

`string`[]

A list of metadata key value pairs to filter by

**Example**

```ts
["key1:value1","key2:value2"]
```

###### workflows?

`string`[]

A list of workflow IDs to filter by

##### params?

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`TenantQueueMetrics`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/TenantQueueMetrics.md), `any`\>\>

#### Description

Get the queue metrics for the tenant

#### Tags

Workflow

#### Name

TenantGetQueueMetrics

#### Request

GET:/api/v1/tenants/{tenant}/queue-metrics

#### Secure

***

### tenantGetStepRunQueueMetrics()

> **tenantGetStepRunQueueMetrics**(`tenant`, `params`): `Promise`\<`AxiosResponse`\<[`TenantStepRunQueueMetrics`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/TenantStepRunQueueMetrics.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1236](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1236)

#### Parameters

##### tenant

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`TenantStepRunQueueMetrics`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/TenantStepRunQueueMetrics.md), `any`\>\>

#### Description

Get the queue metrics for the tenant

#### Tags

Tenant

#### Name

TenantGetStepRunQueueMetrics

#### Request

GET:/api/v1/tenants/{tenant}/step-run-queue-metrics

#### Secure

***

### tenantInviteAccept()

> **tenantInviteAccept**(`data`, `params`): `Promise`\<`AxiosResponse`\<`void`, `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:984](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L984)

#### Parameters

##### data

[`AcceptInviteRequest`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/AcceptInviteRequest.md)

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<`void`, `any`\>\>

#### Description

Accepts a tenant invite

#### Tags

Tenant

#### Name

TenantInviteAccept

#### Request

POST:/api/v1/users/invites/accept

#### Secure

***

### tenantInviteCreate()

> **tenantInviteCreate**(`tenant`, `data`, `params`): `Promise`\<`AxiosResponse`\<[`TenantInvite`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/TenantInvite.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1075](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1075)

#### Parameters

##### tenant

`string`

##### data

[`CreateTenantInviteRequest`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/CreateTenantInviteRequest.md)

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`TenantInvite`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/TenantInvite.md), `any`\>\>

#### Description

Creates a new tenant invite

#### Tags

Tenant

#### Name

TenantInviteCreate

#### Request

POST:/api/v1/tenants/{tenant}/invites

#### Secure

***

### tenantInviteDelete()

> **tenantInviteDelete**(`tenant`, `tenantInvite`, `params`): `Promise`\<`AxiosResponse`\<[`TenantInvite`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/TenantInvite.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1137](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1137)

#### Parameters

##### tenant

`string`

##### tenantInvite

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`TenantInvite`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/TenantInvite.md), `any`\>\>

#### Description

Deletes a tenant invite

#### Name

TenantInviteDelete

#### Request

DELETE:/api/v1/tenants/{tenant}/invites/{tenant-invite}

#### Secure

***

### tenantInviteList()

> **tenantInviteList**(`tenant`, `params`): `Promise`\<`AxiosResponse`\<[`TenantInviteList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/TenantInviteList.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1098](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1098)

#### Parameters

##### tenant

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`TenantInviteList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/TenantInviteList.md), `any`\>\>

#### Description

Gets a list of tenant invites

#### Tags

Tenant

#### Name

TenantInviteList

#### Request

GET:/api/v1/tenants/{tenant}/invites

#### Secure

***

### tenantInviteReject()

> **tenantInviteReject**(`data`, `params`): `Promise`\<`AxiosResponse`\<`void`, `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1002](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1002)

#### Parameters

##### data

[`RejectInviteRequest`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/RejectInviteRequest.md)

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<`void`, `any`\>\>

#### Description

Rejects a tenant invite

#### Tags

Tenant

#### Name

TenantInviteReject

#### Request

POST:/api/v1/users/invites/reject

#### Secure

***

### tenantInviteUpdate()

> **tenantInviteUpdate**(`tenant`, `tenantInvite`, `data`, `params`): `Promise`\<`AxiosResponse`\<[`TenantInvite`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/TenantInvite.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1114](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1114)

#### Parameters

##### tenant

`string`

##### tenantInvite

`string`

##### data

[`UpdateTenantInviteRequest`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/UpdateTenantInviteRequest.md)

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`TenantInvite`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/TenantInvite.md), `any`\>\>

#### Description

Updates a tenant invite

#### Name

TenantInviteUpdate

#### Request

PATCH:/api/v1/tenants/{tenant}/invites/{tenant-invite}

#### Secure

***

### tenantMemberDelete()

> **tenantMemberDelete**(`tenant`, `member`, `params`): `Promise`\<`AxiosResponse`\<[`TenantMember`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/TenantMember.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1442](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1442)

#### Parameters

##### tenant

`string`

##### member

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`TenantMember`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/TenantMember.md), `any`\>\>

#### Description

Delete a member from a tenant

#### Tags

Tenant

#### Name

TenantMemberDelete

#### Request

DELETE:/api/v1/tenants/{tenant}/members/{member}

#### Secure

***

### tenantMemberList()

> **tenantMemberList**(`tenant`, `params`): `Promise`\<`AxiosResponse`\<[`TenantMemberList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/TenantMemberList.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1425](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1425)

#### Parameters

##### tenant

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`TenantMemberList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/TenantMemberList.md), `any`\>\>

#### Description

Gets a list of tenant members

#### Tags

Tenant

#### Name

TenantMemberList

#### Request

GET:/api/v1/tenants/{tenant}/members

#### Secure

***

### tenantMembershipsList()

> **tenantMembershipsList**(`params`): `Promise`\<`AxiosResponse`\<[`UserTenantMembershipsList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/UserTenantMembershipsList.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:950](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L950)

#### Parameters

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`UserTenantMembershipsList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/UserTenantMembershipsList.md), `any`\>\>

#### Description

Lists all tenant memberships for the current user

#### Tags

User

#### Name

TenantMembershipsList

#### Request

GET:/api/v1/users/memberships

#### Secure

***

### tenantResourcePolicyGet()

> **tenantResourcePolicyGet**(`tenant`, `params`): `Promise`\<`AxiosResponse`\<[`TenantResourcePolicy`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/TenantResourcePolicy.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:775](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L775)

#### Parameters

##### tenant

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`TenantResourcePolicy`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/TenantResourcePolicy.md), `any`\>\>

#### Description

Gets the resource policy for a tenant

#### Tags

Tenant

#### Name

TenantResourcePolicyGet

#### Request

GET:/api/v1/tenants/{tenant}/resource-policy

#### Secure

***

### tenantUpdate()

> **tenantUpdate**(`tenant`, `data`, `params`): `Promise`\<`AxiosResponse`\<[`Tenant`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/Tenant.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1039](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1039)

#### Parameters

##### tenant

`string`

##### data

[`UpdateTenantRequest`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/UpdateTenantRequest.md)

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`Tenant`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/Tenant.md), `any`\>\>

#### Description

Update an existing tenant

#### Tags

Tenant

#### Name

TenantUpdate

#### Request

PATCH:/api/v1/tenants/{tenant}

#### Secure

***

### userCreate()

> **userCreate**(`data`, `params`): `Promise`\<`AxiosResponse`\<[`User`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/User.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:915](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L915)

#### Parameters

##### data

[`UserRegisterRequest`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/UserRegisterRequest.md)

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`User`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/User.md), `any`\>\>

#### Description

Registers a user.

#### Tags

User

#### Name

UserCreate

#### Request

POST:/api/v1/users/register

***

### userGetCurrent()

> **userGetCurrent**(`params`): `Promise`\<`AxiosResponse`\<[`User`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/User.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:880](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L880)

#### Parameters

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`User`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/User.md), `any`\>\>

#### Description

Gets the current user

#### Tags

User

#### Name

UserGetCurrent

#### Request

GET:/api/v1/users/current

#### Secure

***

### userListTenantInvites()

> **userListTenantInvites**(`params`): `Promise`\<`AxiosResponse`\<[`TenantInviteList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/TenantInviteList.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:967](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L967)

#### Parameters

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`TenantInviteList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/TenantInviteList.md), `any`\>\>

#### Description

Lists all tenant invites for the current user

#### Tags

Tenant

#### Name

UserListTenantInvites

#### Request

GET:/api/v1/users/invites

#### Secure

***

### userUpdateGithubOauthCallback()

> **userUpdateGithubOauthCallback**(`params`): `Promise`\<`AxiosResponse`\<`any`, `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:638](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L638)

#### Parameters

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<`any`, `any`\>\>

#### Description

Completes the OAuth flow

#### Tags

User

#### Name

UserUpdateGithubOauthCallback

#### Request

GET:/api/v1/users/github/callback

***

### userUpdateGithubOauthStart()

> **userUpdateGithubOauthStart**(`params`): `Promise`\<`AxiosResponse`\<`any`, `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:624](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L624)

#### Parameters

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<`any`, `any`\>\>

#### Description

Starts the OAuth flow

#### Tags

User

#### Name

UserUpdateGithubOauthStart

#### Request

GET:/api/v1/users/github/start

***

### userUpdateGoogleOauthCallback()

> **userUpdateGoogleOauthCallback**(`params`): `Promise`\<`AxiosResponse`\<`any`, `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:610](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L610)

#### Parameters

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<`any`, `any`\>\>

#### Description

Completes the OAuth flow

#### Tags

User

#### Name

UserUpdateGoogleOauthCallback

#### Request

GET:/api/v1/users/google/callback

***

### userUpdateGoogleOauthStart()

> **userUpdateGoogleOauthStart**(`params`): `Promise`\<`AxiosResponse`\<`any`, `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:596](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L596)

#### Parameters

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<`any`, `any`\>\>

#### Description

Starts the OAuth flow

#### Tags

User

#### Name

UserUpdateGoogleOauthStart

#### Request

GET:/api/v1/users/google/start

***

### userUpdateLogin()

> **userUpdateLogin**(`data`, `params`): `Promise`\<`AxiosResponse`\<[`User`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/User.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:579](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L579)

#### Parameters

##### data

[`UserLoginRequest`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/UserLoginRequest.md)

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`User`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/User.md), `any`\>\>

#### Description

Logs in a user.

#### Tags

User

#### Name

UserUpdateLogin

#### Request

POST:/api/v1/users/login

***

### userUpdateLogout()

> **userUpdateLogout**(`params`): `Promise`\<`AxiosResponse`\<[`User`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/User.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:933](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L933)

#### Parameters

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`User`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/User.md), `any`\>\>

#### Description

Logs out a user.

#### Tags

User

#### Name

UserUpdateLogout

#### Request

POST:/api/v1/users/logout

#### Secure

***

### userUpdatePassword()

> **userUpdatePassword**(`data`, `params`): `Promise`\<`AxiosResponse`\<[`User`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/User.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:897](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L897)

#### Parameters

##### data

[`UserChangePasswordRequest`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/UserChangePasswordRequest.md)

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`User`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/User.md), `any`\>\>

#### Description

Update a user password.

#### Tags

User

#### Name

UserUpdatePassword

#### Request

POST:/api/v1/users/password

#### Secure

***

### userUpdateSlackOauthCallback()

> **userUpdateSlackOauthCallback**(`params`): `Promise`\<`AxiosResponse`\<`any`, `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:669](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L669)

#### Parameters

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<`any`, `any`\>\>

#### Description

Completes the OAuth flow

#### Tags

User

#### Name

UserUpdateSlackOauthCallback

#### Request

GET:/api/v1/users/slack/callback

#### Secure

***

### userUpdateSlackOauthStart()

> **userUpdateSlackOauthStart**(`tenant`, `params`): `Promise`\<`AxiosResponse`\<`any`, `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:653](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L653)

#### Parameters

##### tenant

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<`any`, `any`\>\>

#### Description

Starts the OAuth flow

#### Tags

User

#### Name

UserUpdateSlackOauthStart

#### Request

GET:/api/v1/tenants/{tenant}/slack/start

#### Secure

***

### v1DagListTasks()

> **v1DagListTasks**(`query`, `params`): `Promise`\<`AxiosResponse`\<[`V1DagChildren`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/V1DagChildren.md)[], `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:239](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L239)

#### Parameters

##### query

###### dag_ids

`string`[]

The external id of the DAG

###### tenant

`string`

The tenant id

**Format**

uuid

**Min Length**

36

**Max Length**

36

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`V1DagChildren`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/V1DagChildren.md)[], `any`\>\>

#### Description

Lists all tasks that belong a specific list of dags

#### Tags

Task

#### Name

V1DagListTasks

#### Request

GET:/api/v1/stable/dags/tasks

#### Secure

***

### v1LogLineList()

> **v1LogLineList**(`task`, `params`): `Promise`\<`AxiosResponse`\<[`V1LogLineList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/V1LogLineList.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:186](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L186)

#### Parameters

##### task

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`V1LogLineList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/V1LogLineList.md), `any`\>\>

#### Description

Lists log lines for a task

#### Tags

Log

#### Name

V1LogLineList

#### Request

GET:/api/v1/stable/tasks/{task}/logs

#### Secure

***

### v1TaskCancel()

> **v1TaskCancel**(`tenant`, `data`, `params`): `Promise`\<`AxiosResponse`\<`void`, `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:203](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L203)

#### Parameters

##### tenant

`string`

##### data

[`V1CancelTaskRequest`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/V1CancelTaskRequest.md)

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<`void`, `any`\>\>

#### Description

Cancel tasks

#### Tags

Task

#### Name

V1TaskCancel

#### Request

POST:/api/v1/stable/tenants/{tenant}/tasks/cancel

#### Secure

***

### v1TaskEventList()

> **v1TaskEventList**(`task`, `query?`, `params?`): `Promise`\<`AxiosResponse`\<[`V1TaskEventList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/V1TaskEventList.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:153](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L153)

#### Parameters

##### task

`string`

##### query?

###### limit?

`number`

The number to limit by

**Format**

int64

###### offset?

`number`

The number to skip

**Format**

int64

##### params?

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`V1TaskEventList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/V1TaskEventList.md), `any`\>\>

#### Description

List events for a task

#### Tags

Task

#### Name

V1TaskEventList

#### Request

GET:/api/v1/stable/tasks/{task}/task-events

#### Secure

***

### v1TaskGet()

> **v1TaskGet**(`task`, `params`): `Promise`\<`AxiosResponse`\<[`V1TaskSummary`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/V1TaskSummary.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:136](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L136)

#### Parameters

##### task

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`V1TaskSummary`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/V1TaskSummary.md), `any`\>\>

#### Description

Get a task by id

#### Tags

Task

#### Name

V1TaskGet

#### Request

GET:/api/v1/stable/tasks/{task}

#### Secure

***

### v1TaskGetPointMetrics()

> **v1TaskGetPointMetrics**(`tenant`, `query?`, `params?`): `Promise`\<`AxiosResponse`\<[`V1TaskPointMetrics`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/V1TaskPointMetrics.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:470](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L470)

#### Parameters

##### tenant

`string`

##### query?

###### createdAfter?

`string`

The time after the task was created

**Format**

date-time

**Example**

```ts
"2021-01-01T00:00:00Z"
```

###### finishedBefore?

`string`

The time before the task was completed

**Format**

date-time

**Example**

```ts
"2021-01-01T00:00:00Z"
```

##### params?

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`V1TaskPointMetrics`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/V1TaskPointMetrics.md), `any`\>\>

#### Description

Get a minute by minute breakdown of task metrics for a tenant

#### Tags

Task

#### Name

V1TaskGetPointMetrics

#### Request

GET:/api/v1/stable/tenants/{tenant}/task-point-metrics

#### Secure

***

### v1TaskListStatusMetrics()

> **v1TaskListStatusMetrics**(`tenant`, `query`, `params`): `Promise`\<`AxiosResponse`\<[`V1TaskRunMetrics`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/type-aliases/V1TaskRunMetrics.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:433](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L433)

#### Parameters

##### tenant

`string`

##### query

###### parent_task_external_id?

`string`

The parent task's external id

**Format**

uuid

**Min Length**

36

**Max Length**

36

###### since

`string`

The start time to get metrics for

**Format**

date-time

###### workflow_ids?

`string`[]

The workflow id to find runs for

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`V1TaskRunMetrics`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/type-aliases/V1TaskRunMetrics.md), `any`\>\>

#### Description

Get a summary of task run metrics for a tenant

#### Tags

Task

#### Name

V1TaskListStatusMetrics

#### Request

GET:/api/v1/stable/tenants/{tenant}/task-metrics

#### Secure

***

### v1TaskReplay()

> **v1TaskReplay**(`tenant`, `data`, `params`): `Promise`\<`AxiosResponse`\<`void`, `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:221](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L221)

#### Parameters

##### tenant

`string`

##### data

[`V1ReplayTaskRequest`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/V1ReplayTaskRequest.md)

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<`void`, `any`\>\>

#### Description

Replay tasks

#### Tags

Task

#### Name

V1TaskReplay

#### Request

POST:/api/v1/stable/tenants/{tenant}/tasks/replay

#### Secure

***

### v1WorkflowRunCreate()

> **v1WorkflowRunCreate**(`tenant`, `data`, `params`): `Promise`\<`AxiosResponse`\<[`V1WorkflowRunDetails`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/V1WorkflowRunDetails.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:360](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L360)

#### Parameters

##### tenant

`string`

##### data

[`V1TriggerWorkflowRunRequest`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/V1TriggerWorkflowRunRequest.md)

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`V1WorkflowRunDetails`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/V1WorkflowRunDetails.md), `any`\>\>

#### Description

Trigger a new workflow run

#### Tags

Workflow Runs

#### Name

V1WorkflowRunCreate

#### Request

POST:/api/v1/stable/tenants/{tenant}/workflow-runs/trigger

#### Secure

***

### v1WorkflowRunDisplayNamesList()

> **v1WorkflowRunDisplayNamesList**(`tenant`, `query`, `params`): `Promise`\<`AxiosResponse`\<[`V1WorkflowRunDisplayNameList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/V1WorkflowRunDisplayNameList.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:335](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L335)

#### Parameters

##### tenant

`string`

##### query

###### external_ids

`string`[]

The external ids of the workflow runs to get display names for

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`V1WorkflowRunDisplayNameList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/V1WorkflowRunDisplayNameList.md), `any`\>\>

#### Description

Lists displayable names of workflow runs for a tenant

#### Tags

Workflow Runs

#### Name

V1WorkflowRunDisplayNamesList

#### Request

GET:/api/v1/stable/tenants/{tenant}/workflow-runs/display-names

#### Secure

***

### v1WorkflowRunGet()

> **v1WorkflowRunGet**(`v1WorkflowRun`, `params`): `Promise`\<`AxiosResponse`\<[`V1WorkflowRunDetails`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/V1WorkflowRunDetails.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:383](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L383)

#### Parameters

##### v1WorkflowRun

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`V1WorkflowRunDetails`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/V1WorkflowRunDetails.md), `any`\>\>

#### Description

Get a workflow run and its metadata to display on the "detail" page

#### Tags

Workflow Runs

#### Name

V1WorkflowRunGet

#### Request

GET:/api/v1/stable/workflow-runs/{v1-workflow-run}

#### Secure

***

### v1WorkflowRunList()

> **v1WorkflowRunList**(`tenant`, `query`, `params`): `Promise`\<`AxiosResponse`\<[`V1TaskSummaryList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/V1TaskSummaryList.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:270](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L270)

#### Parameters

##### tenant

`string`

##### query

###### additional_metadata?

`string`[]

Additional metadata k-v pairs to filter by

###### limit?

`number`

The number to limit by

**Format**

int64

###### offset?

`number`

The number to skip

**Format**

int64

###### only_tasks

`boolean`

Whether to include DAGs or only to include tasks

###### parent_task_external_id?

`string`

The parent task external id to filter by

**Format**

uuid

**Min Length**

36

**Max Length**

36

###### since

`string`

The earliest date to filter by

**Format**

date-time

###### statuses?

[`V1TaskStatus`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/enumerations/V1TaskStatus.md)[]

A list of statuses to filter by

###### until?

`string`

The latest date to filter by

**Format**

date-time

###### worker_id?

`string`

The worker id to filter by

**Format**

uuid

**Min Length**

36

**Max Length**

36

###### workflow_ids?

`string`[]

The workflow ids to find runs for

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`V1TaskSummaryList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/V1TaskSummaryList.md), `any`\>\>

#### Description

Lists workflow runs for a tenant.

#### Tags

Workflow Runs

#### Name

V1WorkflowRunList

#### Request

GET:/api/v1/stable/tenants/{tenant}/workflow-runs

#### Secure

***

### v1WorkflowRunTaskEventsList()

> **v1WorkflowRunTaskEventsList**(`v1WorkflowRun`, `query?`, `params?`): `Promise`\<`AxiosResponse`\<[`V1TaskEventList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/V1TaskEventList.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:400](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L400)

#### Parameters

##### v1WorkflowRun

`string`

##### query?

###### limit?

`number`

The number to limit by

**Format**

int64

###### offset?

`number`

The number to skip

**Format**

int64

##### params?

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`V1TaskEventList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/V1TaskEventList.md), `any`\>\>

#### Description

List all tasks for a workflow run

#### Tags

Workflow Runs

#### Name

V1WorkflowRunTaskEventsList

#### Request

GET:/api/v1/stable/workflow-runs/{v1-workflow-run}/task-events

#### Secure

***

### webhookCreate()

> **webhookCreate**(`tenant`, `data`, `params`): `Promise`\<`AxiosResponse`\<[`WebhookWorkerCreated`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WebhookWorkerCreated.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:2472](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L2472)

#### Parameters

##### tenant

`string`

##### data

[`WebhookWorkerCreateRequest`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WebhookWorkerCreateRequest.md)

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`WebhookWorkerCreated`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WebhookWorkerCreated.md), `any`\>\>

#### Description

Creates a webhook

#### Name

WebhookCreate

#### Request

POST:/api/v1/tenants/{tenant}/webhook-workers

#### Secure

***

### webhookDelete()

> **webhookDelete**(`webhook`, `params`): `Promise`\<`AxiosResponse`\<`void`, `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:2490](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L2490)

#### Parameters

##### webhook

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<`void`, `any`\>\>

#### Description

Deletes a webhook

#### Name

WebhookDelete

#### Request

DELETE:/api/v1/webhook-workers/{webhook}

#### Secure

***

### webhookList()

> **webhookList**(`tenant`, `params`): `Promise`\<`AxiosResponse`\<[`WebhookWorkerListResponse`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WebhookWorkerListResponse.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:2456](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L2456)

#### Parameters

##### tenant

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`WebhookWorkerListResponse`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WebhookWorkerListResponse.md), `any`\>\>

#### Description

Lists all webhooks

#### Name

WebhookList

#### Request

GET:/api/v1/tenants/{tenant}/webhook-workers

#### Secure

***

### webhookRequestsList()

> **webhookRequestsList**(`webhook`, `params`): `Promise`\<`AxiosResponse`\<[`WebhookWorkerRequestListResponse`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WebhookWorkerRequestListResponse.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:2505](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L2505)

#### Parameters

##### webhook

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`WebhookWorkerRequestListResponse`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WebhookWorkerRequestListResponse.md), `any`\>\>

#### Description

Lists all requests for a webhook

#### Name

WebhookRequestsList

#### Request

GET:/api/v1/webhook-workers/{webhook}/requests

#### Secure

***

### workerGet()

> **workerGet**(`worker`, `params`): `Promise`\<`AxiosResponse`\<[`Worker`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/Worker.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:2440](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L2440)

#### Parameters

##### worker

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`Worker`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/Worker.md), `any`\>\>

#### Description

Get a worker

#### Tags

Worker

#### Name

WorkerGet

#### Request

GET:/api/v1/workers/{worker}

#### Secure

***

### workerList()

> **workerList**(`tenant`, `params`): `Promise`\<`AxiosResponse`\<[`WorkerList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkerList.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:2404](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L2404)

#### Parameters

##### tenant

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`WorkerList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkerList.md), `any`\>\>

#### Description

Get all workers for a tenant

#### Tags

Worker

#### Name

WorkerList

#### Request

GET:/api/v1/tenants/{tenant}/worker

#### Secure

***

### workerUpdate()

> **workerUpdate**(`worker`, `data`, `params`): `Promise`\<`AxiosResponse`\<[`Worker`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/Worker.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:2421](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L2421)

#### Parameters

##### worker

`string`

##### data

[`UpdateWorkerRequest`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/UpdateWorkerRequest.md)

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`Worker`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/Worker.md), `any`\>\>

#### Description

Update a worker

#### Tags

Worker

#### Name

WorkerUpdate

#### Request

PATCH:/api/v1/workers/{worker}

#### Secure

***

### workflowCronDelete()

> **workflowCronDelete**(`tenant`, `cronWorkflow`, `params`): `Promise`\<`AxiosResponse`\<`void`, `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1767](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1767)

#### Parameters

##### tenant

`string`

##### cronWorkflow

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<`void`, `any`\>\>

#### Description

Delete a cron job workflow run for a tenant

#### Tags

Workflow

#### Name

WorkflowCronDelete

#### Request

DELETE:/api/v1/tenants/{tenant}/workflows/crons/{cron-workflow}

#### Secure

***

### workflowCronGet()

> **workflowCronGet**(`tenant`, `cronWorkflow`, `params`): `Promise`\<`AxiosResponse`\<[`CronWorkflows`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/CronWorkflows.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1750](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1750)

#### Parameters

##### tenant

`string`

##### cronWorkflow

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`CronWorkflows`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/CronWorkflows.md), `any`\>\>

#### Description

Get a cron job workflow run for a tenant

#### Tags

Workflow

#### Name

WorkflowCronGet

#### Request

GET:/api/v1/tenants/{tenant}/workflows/crons/{cron-workflow}

#### Secure

***

### workflowDelete()

> **workflowDelete**(`workflow`, `params`): `Promise`\<`AxiosResponse`\<`void`, `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1828](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1828)

#### Parameters

##### workflow

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<`void`, `any`\>\>

#### Description

Delete a workflow for a tenant

#### Tags

Workflow

#### Name

WorkflowDelete

#### Request

DELETE:/api/v1/workflows/{workflow}

#### Secure

***

### workflowGet()

> **workflowGet**(`workflow`, `params`): `Promise`\<`AxiosResponse`\<[`Workflow`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/Workflow.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1811](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1811)

#### Parameters

##### workflow

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`Workflow`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/Workflow.md), `any`\>\>

#### Description

Get a workflow for a tenant

#### Tags

Workflow

#### Name

WorkflowGet

#### Request

GET:/api/v1/workflows/{workflow}

#### Secure

***

### workflowGetMetrics()

> **workflowGetMetrics**(`workflow`, `query?`, `params?`): `Promise`\<`AxiosResponse`\<[`WorkflowMetrics`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowMetrics.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1926](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1926)

#### Parameters

##### workflow

`string`

##### query?

###### groupKey?

`string`

A group key to filter metrics by

###### status?

[`WorkflowRunStatus`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/enumerations/WorkflowRunStatus.md)

A status of workflow run statuses to filter by

##### params?

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`WorkflowMetrics`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowMetrics.md), `any`\>\>

#### Description

Get the metrics for a workflow version

#### Tags

Workflow

#### Name

WorkflowGetMetrics

#### Request

GET:/api/v1/workflows/{workflow}/metrics

#### Secure

***

### workflowGetWorkersCount()

> **workflowGetWorkersCount**(`tenant`, `workflow`, `params`): `Promise`\<`AxiosResponse`\<[`WorkflowWorkersCount`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowWorkersCount.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:2089](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L2089)

#### Parameters

##### tenant

`string`

##### workflow

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`WorkflowWorkersCount`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowWorkersCount.md), `any`\>\>

#### Description

Get a count of the workers available for workflow

#### Tags

Workflow

#### Name

WorkflowGetWorkersCount

#### Request

GET:/api/v1/tenants/{tenant}/workflows/{workflow}/worker-count

#### Secure

***

### workflowList()

> **workflowList**(`tenant`, `query?`, `params?`): `Promise`\<`AxiosResponse`\<[`WorkflowList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowList.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1510](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1510)

#### Parameters

##### tenant

`string`

##### query?

###### limit?

`number`

The number to limit by

**Format**

int

**Default**

```ts
50
```

###### name?

`string`

Search by name

###### offset?

`number`

The number to skip

**Format**

int

**Default**

```ts
0
```

##### params?

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`WorkflowList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowList.md), `any`\>\>

#### Description

Get all workflows for a tenant

#### Tags

Workflow

#### Name

WorkflowList

#### Request

GET:/api/v1/tenants/{tenant}/workflows

#### Secure

***

### workflowRunCancel()

> **workflowRunCancel**(`tenant`, `data`, `params`): `Promise`\<`AxiosResponse`\<\{ `workflowRunIds`: `string`[]; \}, `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1783](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1783)

#### Parameters

##### tenant

`string`

##### data

[`WorkflowRunsCancelRequest`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowRunsCancelRequest.md)

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<\{ `workflowRunIds`: `string`[]; \}, `any`\>\>

#### Description

Cancel a batch of workflow runs

#### Tags

Workflow Run

#### Name

WorkflowRunCancel

#### Request

POST:/api/v1/tenants/{tenant}/workflows/cancel

#### Secure

***

### workflowRunCreate()

> **workflowRunCreate**(`workflow`, `data`, `query?`, `params?`): `Promise`\<`AxiosResponse`\<[`WorkflowRun`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowRun.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1893](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1893)

#### Parameters

##### workflow

`string`

##### data

[`TriggerWorkflowRunRequest`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/TriggerWorkflowRunRequest.md)

##### query?

###### version?

`string`

The workflow version. If not supplied, the latest version is fetched.

**Format**

uuid

**Min Length**

36

**Max Length**

36

##### params?

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`WorkflowRun`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowRun.md), `any`\>\>

#### Description

Trigger a new workflow run for a tenant

#### Tags

Workflow Run

#### Name

WorkflowRunCreate

#### Request

POST:/api/v1/workflows/{workflow}/trigger

#### Secure

***

### workflowRunGet()

> **workflowRunGet**(`tenant`, `workflowRun`, `params`): `Promise`\<`AxiosResponse`\<[`WorkflowRun`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowRun.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:2295](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L2295)

#### Parameters

##### tenant

`string`

##### workflowRun

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`WorkflowRun`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowRun.md), `any`\>\>

#### Description

Get a workflow run for a tenant

#### Tags

Workflow

#### Name

WorkflowRunGet

#### Request

GET:/api/v1/tenants/{tenant}/workflow-runs/{workflow-run}

#### Secure

***

### workflowRunGetInput()

> **workflowRunGetInput**(`tenant`, `workflowRun`, `params`): `Promise`\<`AxiosResponse`\<`Record`\<`string`, `any`\>, `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:2522](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L2522)

#### Parameters

##### tenant

`string`

##### workflowRun

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<`Record`\<`string`, `any`\>, `any`\>\>

#### Description

Get the input for a workflow run.

#### Tags

Workflow Run

#### Name

WorkflowRunGetInput

#### Request

GET:/api/v1/tenants/{tenant}/workflow-runs/{workflow-run}/input

#### Secure

***

### workflowRunGetMetrics()

> **workflowRunGetMetrics**(`tenant`, `query?`, `params?`): `Promise`\<`AxiosResponse`\<[`WorkflowRunsMetrics`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowRunsMetrics.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:2227](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L2227)

#### Parameters

##### tenant

`string`

##### query?

###### additionalMetadata?

`string`[]

A list of metadata key value pairs to filter by

**Example**

```ts
["key1:value1","key2:value2"]
```

###### createdAfter?

`string`

The time after the workflow run was created

**Format**

date-time

**Example**

```ts
"2021-01-01T00:00:00Z"
```

###### createdBefore?

`string`

The time before the workflow run was created

**Format**

date-time

**Example**

```ts
"2021-01-01T00:00:00Z"
```

###### eventId?

`string`

The event id to get runs for.

**Format**

uuid

**Min Length**

36

**Max Length**

36

###### parentStepRunId?

`string`

The parent step run id

**Format**

uuid

**Min Length**

36

**Max Length**

36

###### parentWorkflowRunId?

`string`

The parent workflow run id

**Format**

uuid

**Min Length**

36

**Max Length**

36

###### workflowId?

`string`

The workflow id to get runs for.

**Format**

uuid

**Min Length**

36

**Max Length**

36

##### params?

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`WorkflowRunsMetrics`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowRunsMetrics.md), `any`\>\>

#### Description

Get a summary of  workflow run metrics for a tenant

#### Tags

Workflow

#### Name

WorkflowRunGetMetrics

#### Request

GET:/api/v1/tenants/{tenant}/workflows/runs/metrics

#### Secure

***

### workflowRunGetShape()

> **workflowRunGetShape**(`tenant`, `workflowRun`, `params`): `Promise`\<`AxiosResponse`\<[`WorkflowRunShape`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowRunShape.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:2312](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L2312)

#### Parameters

##### tenant

`string`

##### workflowRun

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`WorkflowRunShape`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowRunShape.md), `any`\>\>

#### Description

Get a workflow run for a tenant

#### Tags

Workflow

#### Name

WorkflowRunGetShape

#### Request

GET:/api/v1/tenants/{tenant}/workflow-runs/{workflow-run}/shape

#### Secure

***

### workflowRunList()

> **workflowRunList**(`tenant`, `query?`, `params?`): `Promise`\<`AxiosResponse`\<[`WorkflowRunList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowRunList.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:2106](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L2106)

#### Parameters

##### tenant

`string`

##### query?

###### additionalMetadata?

`string`[]

A list of metadata key value pairs to filter by

**Example**

```ts
["key1:value1","key2:value2"]
```

###### createdAfter?

`string`

The time after the workflow run was created

**Format**

date-time

**Example**

```ts
"2021-01-01T00:00:00Z"
```

###### createdBefore?

`string`

The time before the workflow run was created

**Format**

date-time

**Example**

```ts
"2021-01-01T00:00:00Z"
```

###### eventId?

`string`

The event id to get runs for.

**Format**

uuid

**Min Length**

36

**Max Length**

36

###### finishedAfter?

`string`

The time after the workflow run was finished

**Format**

date-time

**Example**

```ts
"2021-01-01T00:00:00Z"
```

###### finishedBefore?

`string`

The time before the workflow run was finished

**Format**

date-time

**Example**

```ts
"2021-01-01T00:00:00Z"
```

###### kinds?

[`WorkflowKindList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/type-aliases/WorkflowKindList.md)

A list of workflow kinds to filter by

###### limit?

`number`

The number to limit by

**Format**

int64

###### offset?

`number`

The number to skip

**Format**

int64

###### orderByDirection?

[`WorkflowRunOrderByDirection`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/enumerations/WorkflowRunOrderByDirection.md)

The order by direction

###### orderByField?

[`WorkflowRunOrderByField`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/enumerations/WorkflowRunOrderByField.md)

The order by field

###### parentStepRunId?

`string`

The parent step run id

**Format**

uuid

**Min Length**

36

**Max Length**

36

###### parentWorkflowRunId?

`string`

The parent workflow run id

**Format**

uuid

**Min Length**

36

**Max Length**

36

###### statuses?

[`WorkflowRunStatusList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/type-aliases/WorkflowRunStatusList.md)

A list of workflow run statuses to filter by

###### workflowId?

`string`

The workflow id to get runs for.

**Format**

uuid

**Min Length**

36

**Max Length**

36

##### params?

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`WorkflowRunList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowRunList.md), `any`\>\>

#### Description

Get all workflow runs for a tenant

#### Tags

Workflow

#### Name

WorkflowRunList

#### Request

GET:/api/v1/tenants/{tenant}/workflows/runs

#### Secure

***

### workflowRunListStepRunEvents()

> **workflowRunListStepRunEvents**(`tenant`, `workflowRun`, `query?`, `params?`): `Promise`\<`AxiosResponse`\<[`StepRunEventList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/StepRunEventList.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:2027](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L2027)

#### Parameters

##### tenant

`string`

##### workflowRun

`string`

##### query?

###### lastId?

`number`

Last ID of the last event

**Format**

int32

##### params?

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`StepRunEventList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/StepRunEventList.md), `any`\>\>

#### Description

List events for all step runs for a workflow run

#### Tags

Step Run

#### Name

WorkflowRunListStepRunEvents

#### Request

GET:/api/v1/tenants/{tenant}/workflow-runs/{workflow-run}/step-run-events

#### Secure

***

### workflowRunUpdateReplay()

> **workflowRunUpdateReplay**(`tenant`, `data`, `params`): `Promise`\<`AxiosResponse`\<[`ReplayWorkflowRunsResponse`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/ReplayWorkflowRunsResponse.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:2204](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L2204)

#### Parameters

##### tenant

`string`

##### data

[`ReplayWorkflowRunsRequest`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/ReplayWorkflowRunsRequest.md)

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`ReplayWorkflowRunsResponse`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/ReplayWorkflowRunsResponse.md), `any`\>\>

#### Description

Replays a list of workflow runs.

#### Tags

Workflow Run

#### Name

WorkflowRunUpdateReplay

#### Request

POST:/api/v1/tenants/{tenant}/workflow-runs/replay

#### Secure

***

### workflowScheduledDelete()

> **workflowScheduledDelete**(`tenant`, `scheduledWorkflowRun`, `params`): `Promise`\<`AxiosResponse`\<`void`, `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1657](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1657)

#### Parameters

##### tenant

`string`

##### scheduledWorkflowRun

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<`void`, `any`\>\>

#### Description

Delete a scheduled workflow run for a tenant

#### Tags

Workflow

#### Name

WorkflowScheduledDelete

#### Request

DELETE:/api/v1/tenants/{tenant}/workflows/scheduled/{scheduled-workflow-run}

#### Secure

***

### workflowScheduledGet()

> **workflowScheduledGet**(`tenant`, `scheduledWorkflowRun`, `params`): `Promise`\<`AxiosResponse`\<[`ScheduledWorkflows`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/ScheduledWorkflows.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1636](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1636)

#### Parameters

##### tenant

`string`

##### scheduledWorkflowRun

`string`

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`ScheduledWorkflows`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/ScheduledWorkflows.md), `any`\>\>

#### Description

Get a scheduled workflow run for a tenant

#### Tags

Workflow

#### Name

WorkflowScheduledGet

#### Request

GET:/api/v1/tenants/{tenant}/workflows/scheduled/{scheduled-workflow-run}

#### Secure

***

### workflowScheduledList()

> **workflowScheduledList**(`tenant`, `query?`, `params?`): `Promise`\<`AxiosResponse`\<[`ScheduledWorkflowsList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/ScheduledWorkflowsList.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1571](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1571)

#### Parameters

##### tenant

`string`

##### query?

###### additionalMetadata?

`string`[]

A list of metadata key value pairs to filter by

**Example**

```ts
["key1:value1","key2:value2"]
```

###### limit?

`number`

The number to limit by

**Format**

int64

###### offset?

`number`

The number to skip

**Format**

int64

###### orderByDirection?

[`WorkflowRunOrderByDirection`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/enumerations/WorkflowRunOrderByDirection.md)

The order by direction

###### orderByField?

[`ScheduledWorkflowsOrderByField`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/enumerations/ScheduledWorkflowsOrderByField.md)

The order by field

###### parentStepRunId?

`string`

The parent step run id

**Format**

uuid

**Min Length**

36

**Max Length**

36

###### parentWorkflowRunId?

`string`

The parent workflow run id

**Format**

uuid

**Min Length**

36

**Max Length**

36

###### statuses?

[`ScheduledRunStatus`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/enumerations/ScheduledRunStatus.md)[]

A list of scheduled run statuses to filter by

###### workflowId?

`string`

The workflow id to get runs for.

**Format**

uuid

**Min Length**

36

**Max Length**

36

##### params?

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`ScheduledWorkflowsList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/ScheduledWorkflowsList.md), `any`\>\>

#### Description

Get all scheduled workflow runs for a tenant

#### Tags

Workflow

#### Name

WorkflowScheduledList

#### Request

GET:/api/v1/tenants/{tenant}/workflows/scheduled

#### Secure

***

### workflowUpdate()

> **workflowUpdate**(`workflow`, `data`, `params`): `Promise`\<`AxiosResponse`\<[`Workflow`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/Workflow.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1844](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1844)

#### Parameters

##### workflow

`string`

##### data

[`WorkflowUpdateRequest`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowUpdateRequest.md)

##### params

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`Workflow`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/Workflow.md), `any`\>\>

#### Description

Update a workflow for a tenant

#### Tags

Workflow

#### Name

WorkflowUpdate

#### Request

PATCH:/api/v1/workflows/{workflow}

#### Secure

***

### workflowVersionGet()

> **workflowVersionGet**(`workflow`, `query?`, `params?`): `Promise`\<`AxiosResponse`\<[`WorkflowVersion`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowVersion.md), `any`\>\>

Defined in: [src/clients/rest/generated/Api.ts:1863](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/Api.ts#L1863)

#### Parameters

##### workflow

`string`

##### query?

###### version?

`string`

The workflow version. If not supplied, the latest version is fetched.

**Format**

uuid

**Min Length**

36

**Max Length**

36

##### params?

`RequestParams` = `{}`

#### Returns

`Promise`\<`AxiosResponse`\<[`WorkflowVersion`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowVersion.md), `any`\>\>

#### Description

Get a workflow version for a tenant

#### Tags

Workflow

#### Name

WorkflowVersionGet

#### Request

GET:/api/v1/workflows/{workflow}/versions

#### Secure
