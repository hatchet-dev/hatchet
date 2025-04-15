[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / AdminClient

# Class: AdminClient

Defined in: [src/clients/admin/admin-client.ts:48](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/admin/admin-client.ts#L48)

## Constructors

### Constructor

> **new AdminClient**(`config`, `channel`, `factory`, `api`, `tenantId`, `listenerClient`, `workflows`): `AdminClient`

Defined in: [src/clients/admin/admin-client.ts:58](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/admin/admin-client.ts#L58)

#### Parameters

##### config

`ClientConfig`

##### channel

`ChannelImplementation`

##### factory

`ClientFactory`

##### api

[`Api`](Api.md)

##### tenantId

`string`

##### listenerClient

`RunListenerClient`

##### workflows

`undefined` | [`RunsClient`](RunsClient.md)

#### Returns

`AdminClient`

## Properties

### api

> **api**: [`Api`](Api.md)

Defined in: [src/clients/admin/admin-client.ts:52](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/admin/admin-client.ts#L52)

***

### client

> **client**: `WorkflowServiceClient`

Defined in: [src/clients/admin/admin-client.ts:50](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/admin/admin-client.ts#L50)

***

### config

> **config**: `ClientConfig`

Defined in: [src/clients/admin/admin-client.ts:49](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/admin/admin-client.ts#L49)

***

### listenerClient

> **listenerClient**: `RunListenerClient`

Defined in: [src/clients/admin/admin-client.ts:55](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/admin/admin-client.ts#L55)

***

### logger

> **logger**: `Logger`

Defined in: [src/clients/admin/admin-client.ts:54](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/admin/admin-client.ts#L54)

***

### tenantId

> **tenantId**: `string`

Defined in: [src/clients/admin/admin-client.ts:53](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/admin/admin-client.ts#L53)

***

### v1Client

> **v1Client**: `AdminServiceClient`

Defined in: [src/clients/admin/admin-client.ts:51](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/admin/admin-client.ts#L51)

***

### workflows

> **workflows**: `undefined` \| [`RunsClient`](RunsClient.md)

Defined in: [src/clients/admin/admin-client.ts:56](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/admin/admin-client.ts#L56)

## Methods

### ~~get\_workflow()~~

> **get\_workflow**(`workflowId`): `Promise`\<[`Workflow`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/Workflow.md)\>

Defined in: [src/clients/admin/admin-client.ts:290](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/admin/admin-client.ts#L290)

#### Parameters

##### workflowId

`string`

#### Returns

`Promise`\<[`Workflow`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/Workflow.md)\>

#### Deprecated

use getWorkflow instead

***

### ~~get\_workflow\_metrics()~~

> **get\_workflow\_metrics**(`data`): `Promise`\<`AxiosResponse`\<[`WorkflowMetrics`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowMetrics.md), `any`\>\>

Defined in: [src/clients/admin/admin-client.ts:416](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/admin/admin-client.ts#L416)

#### Parameters

##### data

`WorkflowMetricsQuery`

#### Returns

`Promise`\<`AxiosResponse`\<[`WorkflowMetrics`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowMetrics.md), `any`\>\>

#### Deprecated

use getWorkflowMetrics instead

***

### ~~get\_workflow\_run()~~

> **get\_workflow\_run**(`workflowRunId`): `Promise`\<`WorkflowRunRef`\<`unknown`\>\>

Defined in: [src/clients/admin/admin-client.ts:328](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/admin/admin-client.ts#L328)

#### Parameters

##### workflowRunId

`string`

#### Returns

`Promise`\<`WorkflowRunRef`\<`unknown`\>\>

#### Deprecated

use getWorkflowRun instead

***

### ~~get\_workflow\_version()~~

> **get\_workflow\_version**(`workflowId`, `version?`): `Promise`\<[`WorkflowVersion`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowVersion.md)\>

Defined in: [src/clients/admin/admin-client.ts:307](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/admin/admin-client.ts#L307)

#### Parameters

##### workflowId

`string`

##### version?

`string`

#### Returns

`Promise`\<[`WorkflowVersion`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowVersion.md)\>

#### Deprecated

use getWorkflowVersion instead

***

### getWorkflow()

> **getWorkflow**(`workflowId`): `Promise`\<[`Workflow`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/Workflow.md)\>

Defined in: [src/clients/admin/admin-client.ts:299](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/admin/admin-client.ts#L299)

Get a workflow by its ID.

#### Parameters

##### workflowId

`string`

the workflow ID (**note:** this is not the same as the workflow version id)

#### Returns

`Promise`\<[`Workflow`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/Workflow.md)\>

***

### getWorkflowMetrics()

> **getWorkflowMetrics**(`__namedParameters`): `Promise`\<`AxiosResponse`\<[`WorkflowMetrics`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowMetrics.md), `any`\>\>

Defined in: [src/clients/admin/admin-client.ts:427](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/admin/admin-client.ts#L427)

Get the metrics for a workflow.

#### Parameters

##### \_\_namedParameters

`WorkflowMetricsQuery`

#### Returns

`Promise`\<`AxiosResponse`\<[`WorkflowMetrics`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowMetrics.md), `any`\>\>

***

### getWorkflowRun()

> **getWorkflowRun**(`workflowRunId`): `Promise`\<`WorkflowRunRef`\<`unknown`\>\>

Defined in: [src/clients/admin/admin-client.ts:337](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/admin/admin-client.ts#L337)

Get a workflow run.

#### Parameters

##### workflowRunId

`string`

the id of the workflow run to get

#### Returns

`Promise`\<`WorkflowRunRef`\<`unknown`\>\>

the workflow run

***

### getWorkflowVersion()

> **getWorkflowVersion**(`workflowId`, `version?`): `Promise`\<[`WorkflowVersion`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowVersion.md)\>

Defined in: [src/clients/admin/admin-client.ts:317](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/admin/admin-client.ts#L317)

Get a workflow version.

#### Parameters

##### workflowId

`string`

the workflow ID

##### version?

`string`

the version of the workflow to get. If not provided, the latest version will be returned.

#### Returns

`Promise`\<[`WorkflowVersion`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowVersion.md)\>

the workflow version

***

### ~~list\_workflow\_runs()~~

> **list\_workflow\_runs**(`query`): `Promise`\<[`WorkflowRunList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowRunList.md)\>

Defined in: [src/clients/admin/admin-client.ts:344](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/admin/admin-client.ts#L344)

#### Parameters

##### query

###### additionalMetadata?

`string`[]

###### eventId?

`string`

###### limit?

`number`

###### offset?

`number`

###### parentStepRunId?

`string`

###### parentWorkflowRunId?

`string`

###### statuses?

[`WorkflowRunStatusList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/type-aliases/WorkflowRunStatusList.md)

###### workflowId?

`string`

#### Returns

`Promise`\<[`WorkflowRunList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowRunList.md)\>

#### Deprecated

use listWorkflowRuns instead

***

### ~~list\_workflows()~~

> **list\_workflows**(): `Promise`\<[`WorkflowList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowList.md)\>

Defined in: [src/clients/admin/admin-client.ts:274](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/admin/admin-client.ts#L274)

#### Returns

`Promise`\<[`WorkflowList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowList.md)\>

#### Deprecated

use listWorkflows instead

***

### listWorkflowRuns()

> **listWorkflowRuns**(`query`): `Promise`\<[`WorkflowRunList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowRunList.md)\>

Defined in: [src/clients/admin/admin-client.ts:362](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/admin/admin-client.ts#L362)

List workflow runs in the tenant associated with the API token.

#### Parameters

##### query

the query to filter the list of workflow runs

###### additionalMetadata?

`string`[]

###### eventId?

`string`

###### limit?

`number`

###### offset?

`number`

###### parentStepRunId?

`string`

###### parentWorkflowRunId?

`string`

###### statuses?

[`WorkflowRunStatusList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/type-aliases/WorkflowRunStatusList.md)

###### workflowId?

`string`

#### Returns

`Promise`\<[`WorkflowRunList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowRunList.md)\>

***

### listWorkflows()

> **listWorkflows**(): `Promise`\<[`WorkflowList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowList.md)\>

Defined in: [src/clients/admin/admin-client.ts:282](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/admin/admin-client.ts#L282)

List workflows in the tenant associated with the API token.

#### Returns

`Promise`\<[`WorkflowList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowList.md)\>

a list of all workflows in the tenant

***

### ~~put\_rate\_limit()~~

> **put\_rate\_limit**(`key`, `limit`, `duration`): `Promise`\<`void`\>

Defined in: [src/clients/admin/admin-client.ts:113](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/admin/admin-client.ts#L113)

#### Parameters

##### key

`string`

##### limit

`number`

##### duration

[`RateLimitDuration`](../enumerations/RateLimitDuration.md)

#### Returns

`Promise`\<`void`\>

#### Deprecated

use hatchet.ratelimits.upsert instead

***

### ~~put\_workflow()~~

> **put\_workflow**(`opts`): `Promise`\<`WorkflowVersion`\>

Defined in: [src/clients/admin/admin-client.ts:80](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/admin/admin-client.ts#L80)

#### Parameters

##### opts

`CreateWorkflowVersionOpts`

#### Returns

`Promise`\<`WorkflowVersion`\>

#### Deprecated

use putWorkflow instead

***

### ~~putRateLimit()~~

> **putRateLimit**(`key`, `limit`, `duration`): `Promise`\<`void`\>

Defined in: [src/clients/admin/admin-client.ts:120](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/admin/admin-client.ts#L120)

#### Parameters

##### key

`string`

##### limit

`number`

##### duration

[`RateLimitDuration`](../enumerations/RateLimitDuration.md) = `RateLimitDuration.SECOND`

#### Returns

`Promise`\<`void`\>

#### Deprecated

use hatchet.ratelimits.upsert instead

***

### putWorkflow()

> **putWorkflow**(`workflow`): `Promise`\<`WorkflowVersion`\>

Defined in: [src/clients/admin/admin-client.ts:89](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/admin/admin-client.ts#L89)

Creates a new workflow or updates an existing workflow. If the workflow already exists, Hatchet will automatically
determine if the workflow definition has changed and create a new version if necessary.

#### Parameters

##### workflow

`CreateWorkflowVersionOpts`

a workflow definition to create

#### Returns

`Promise`\<`WorkflowVersion`\>

***

### putWorkflowV1()

> **putWorkflowV1**(`workflow`): `Promise`\<`CreateWorkflowVersionResponse`\>

Defined in: [src/clients/admin/admin-client.ts:102](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/admin/admin-client.ts#L102)

Creates a new workflow or updates an existing workflow. If the workflow already exists, Hatchet will automatically
determine if the workflow definition has changed and create a new version if necessary.

#### Parameters

##### workflow

`CreateWorkflowVersionRequest`

a workflow definition to create

#### Returns

`Promise`\<`CreateWorkflowVersionResponse`\>

***

### registerWebhook()

> **registerWebhook**(`data`): `Promise`\<`AxiosResponse`\<[`WebhookWorkerCreated`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WebhookWorkerCreated.md), `any`\>\>

Defined in: [src/clients/admin/admin-client.ts:140](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/admin/admin-client.ts#L140)

#### Parameters

##### data

[`WebhookWorkerCreateRequest`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WebhookWorkerCreateRequest.md)

#### Returns

`Promise`\<`AxiosResponse`\<[`WebhookWorkerCreated`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WebhookWorkerCreated.md), `any`\>\>

***

### ~~run\_workflow()~~

> **run\_workflow**\<`T`\>(`workflowName`, `input`, `options?`): `Promise`\<`WorkflowRunRef`\<`object`\>\>

Defined in: [src/clients/admin/admin-client.ts:147](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/admin/admin-client.ts#L147)

#### Type Parameters

##### T

`T` = `object`

#### Parameters

##### workflowName

`string`

##### input

`T`

##### options?

###### additionalMetadata?

`Record`\<`string`, `string`\>

###### childIndex?

`number`

###### childKey?

`string`

###### parentId?

`string`

###### parentStepRunId?

`string`

#### Returns

`Promise`\<`WorkflowRunRef`\<`object`\>\>

#### Deprecated

use runWorkflow instead

***

### runWorkflow()

> **runWorkflow**\<`Q`, `P`\>(`workflowName`, `input`, `options?`): `WorkflowRunRef`\<`P`\>

Defined in: [src/clients/admin/admin-client.ts:169](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/admin/admin-client.ts#L169)

Run a new instance of a workflow with the given input. This will create a new workflow run and return the ID of the
new run.

#### Type Parameters

##### Q

`Q` = `object`

##### P

`P` = `object`

#### Parameters

##### workflowName

`string`

the name of the workflow to run

##### input

`Q`

an object containing the input to the workflow

##### options?

an object containing the options to run the workflow

###### additionalMetadata?

`Record`\<`string`, `string`\>

###### childIndex?

`number`

###### childKey?

`string`

###### desiredWorkerId?

`string`

###### parentId?

`string`

###### parentStepRunId?

`string`

###### priority?

[`Priority`](../enumerations/Priority.md)

#### Returns

`WorkflowRunRef`\<`P`\>

the ID of the new workflow run

***

### runWorkflows()

> **runWorkflows**\<`Q`, `P`\>(`workflowRuns`): `Promise`\<`WorkflowRunRef`\<`P`\>[]\>

Defined in: [src/clients/admin/admin-client.ts:212](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/admin/admin-client.ts#L212)

Run multiple workflows runs with the given input and options. This will create new workflow runs and return their IDs.
Order is preserved in the response.

#### Type Parameters

##### Q

`Q` = `object`

##### P

`P` = `object`

#### Parameters

##### workflowRuns

`object`[]

an array of objects containing the workflow name, input, and options for each workflow run

#### Returns

`Promise`\<`WorkflowRunRef`\<`P`\>[]\>

an array of workflow run references

***

### ~~schedule\_workflow()~~

> **schedule\_workflow**(`name`, `options?`): `Promise`\<`void`\>

Defined in: [src/clients/admin/admin-client.ts:379](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/admin/admin-client.ts#L379)

#### Parameters

##### name

`string`

##### options?

###### input?

`object`

###### schedules?

`Date`[]

#### Returns

`Promise`\<`void`\>

#### Deprecated

use scheduleWorkflow instead

***

### scheduleWorkflow()

> **scheduleWorkflow**(`name`, `options?`): `void`

Defined in: [src/clients/admin/admin-client.ts:389](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/admin/admin-client.ts#L389)

Schedule a workflow to run at a specific time or times.

#### Parameters

##### name

`string`

the name of the workflow to schedule

##### options?

an object containing the schedules to set

###### input?

`object`

###### schedules?

`Date`[]

#### Returns

`void`
