[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / HatchetClient

# Class: HatchetClient

Defined in: [src/v1/client/client.ts:42](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/client.ts#L42)

HatchetV1 implements the main client interface for interacting with the Hatchet workflow engine.
It provides methods for creating and executing workflows, as well as managing workers.

## Implements

- `IHatchetClient`

## Constructors

### Constructor

> **new HatchetClient**(`config?`, `options?`, `axiosConfig?`): `HatchetClient`

Defined in: [src/v1/client/client.ts:69](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/client.ts#L69)

Creates a new Hatchet client instance.

#### Parameters

##### config?

`Partial`\<`ClientConfig`\>

Optional configuration for the client

##### options?

`HatchetClientOptions`

Optional client options

##### axiosConfig?

`AxiosRequestConfig`\<`any`\>

Optional Axios configuration for HTTP requests

#### Returns

`HatchetClient`

## Properties

### \_api

> **\_api**: [`Api`](Api.md)

Defined in: [src/v1/client/client.ts:45](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/client.ts#L45)

***

### \_isV1

> **\_isV1**: `undefined` \| `boolean` = `true`

Defined in: [src/v1/client/client.ts:57](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/client.ts#L57)

***

### \_v0

> **\_v0**: `InternalHatchetClient`

Defined in: [src/v1/client/client.ts:44](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/client.ts#L44)

The underlying v0 client instance

#### Implementation of

`IHatchetClient._v0`

***

### tenantId

> **tenantId**: `string`

Defined in: [src/v1/client/client.ts:55](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/client.ts#L55)

The tenant ID for the Hatchet client

## Accessors

### admin

#### Get Signature

> **get** **admin**(): [`AdminClient`](AdminClient.md)

Defined in: [src/v1/client/client.ts:418](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/client.ts#L418)

##### Deprecated

use workflow.run, client.run, or client.* feature methods instead

##### Returns

[`AdminClient`](AdminClient.md)

***

### api

#### Get Signature

> **get** **api**(): [`Api`](Api.md)\<`unknown`\>

Defined in: [src/v1/client/client.ts:411](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/client.ts#L411)

Get the API client for making HTTP requests to the Hatchet API
Note: This is not recommended for general use, but is available for advanced scenarios

##### Returns

[`Api`](Api.md)\<`unknown`\>

A API client instance

***

### cron

#### Get Signature

> **get** **cron**(): `CronClient`

Defined in: [src/v1/client/client.ts:295](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/client.ts#L295)

Get the cron client for creating and managing cron workflow runs

##### Deprecated

use client.crons instead

##### Returns

`CronClient`

A cron client instance

***

### crons

#### Get Signature

> **get** **crons**(): `CronClient`

Defined in: [src/v1/client/client.ts:286](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/client.ts#L286)

Get the cron client for creating and managing cron workflow runs

##### Returns

`CronClient`

A cron client instance

***

### event

#### Get Signature

> **get** **event**(): `EventClient`

Defined in: [src/v1/client/client.ts:329](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/client.ts#L329)

Get the event client for creating and managing event workflow runs

##### Deprecated

use client.events instead

##### Returns

`EventClient`

A event client instance

***

### events

#### Get Signature

> **get** **events**(): `EventClient`

Defined in: [src/v1/client/client.ts:320](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/client.ts#L320)

Get the event client for creating and managing event workflow runs

##### Returns

`EventClient`

A event client instance

***

### isV1

#### Get Signature

> **get** **isV1**(): `boolean`

Defined in: [src/v1/client/client.ts:59](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/client.ts#L59)

##### Returns

`boolean`

***

### metrics

#### Get Signature

> **get** **metrics**(): [`MetricsClient`](MetricsClient.md)

Defined in: [src/v1/client/client.ts:339](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/client.ts#L339)

Get the metrics client for creating and managing metrics

##### Returns

[`MetricsClient`](MetricsClient.md)

A metrics client instance

#### Implementation of

`IHatchetClient.metrics`

***

### ratelimits

#### Get Signature

> **get** **ratelimits**(): [`RatelimitsClient`](RatelimitsClient.md)

Defined in: [src/v1/client/client.ts:352](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/client.ts#L352)

Get the rate limits client for creating and managing rate limits

##### Returns

[`RatelimitsClient`](RatelimitsClient.md)

A rate limits client instance

***

### runs

#### Get Signature

> **get** **runs**(): [`RunsClient`](RunsClient.md)

Defined in: [src/v1/client/client.ts:365](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/client.ts#L365)

Get the runs client for creating and managing runs

##### Returns

[`RunsClient`](RunsClient.md)

A runs client instance

#### Implementation of

`IHatchetClient.runs`

***

### schedule

#### Get Signature

> **get** **schedule**(): `ScheduleClient`

Defined in: [src/v1/client/client.ts:312](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/client.ts#L312)

Get the schedule client for creating and managing scheduled workflow runs

##### Deprecated

use client.schedules instead

##### Returns

`ScheduleClient`

A schedule client instance

***

### schedules

#### Get Signature

> **get** **schedules**(): `ScheduleClient`

Defined in: [src/v1/client/client.ts:303](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/client.ts#L303)

Get the schedules client for creating and managing scheduled workflow runs

##### Returns

`ScheduleClient`

A schedules client instance

***

### tasks

#### Get Signature

> **get** **tasks**(): [`WorkflowsClient`](WorkflowsClient.md)

Defined in: [src/v1/client/client.ts:389](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/client.ts#L389)

Get the tasks client for creating and managing tasks

##### Returns

[`WorkflowsClient`](WorkflowsClient.md)

A tasks client instance

***

### v0

#### Get Signature

> **get** **v0**(): `InternalHatchetClient`

Defined in: [src/v1/client/client.ts:50](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/client.ts#L50)

##### Deprecated

v0 client will be removed in a future release, please upgrade to v1

##### Returns

`InternalHatchetClient`

***

### workers

#### Get Signature

> **get** **workers**(): [`WorkersClient`](WorkersClient.md)

Defined in: [src/v1/client/client.ts:399](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/client.ts#L399)

Get the workers client for creating and managing workers

##### Returns

[`WorkersClient`](WorkersClient.md)

A workers client instance

#### Implementation of

`IHatchetClient.workers`

***

### workflows

#### Get Signature

> **get** **workflows**(): [`WorkflowsClient`](WorkflowsClient.md)

Defined in: [src/v1/client/client.ts:378](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/client.ts#L378)

Get the workflows client for creating and managing workflows

##### Returns

[`WorkflowsClient`](WorkflowsClient.md)

A workflows client instance

#### Implementation of

`IHatchetClient.workflows`

## Methods

### durableTask()

Implementation of the durableTask method.

#### Call Signature

> **durableTask**\<`I`, `O`\>(`options`): [`TaskWorkflowDeclaration`](TaskWorkflowDeclaration.md)\<`I`, `O`\>

Defined in: [src/v1/client/client.ts:187](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/client.ts#L187)

Creates a new durable task workflow.
Types can be explicitly specified as generics or inferred from the function signature.

##### Type Parameters

###### I

`I` *extends* [`JsonObject`](../type-aliases/JsonObject.md)

The input type for the durable task

###### O

`O` *extends* [`OutputType`](../type-aliases/OutputType.md)

The output type of the durable task

##### Parameters

###### options

[`CreateDurableTaskWorkflowOpts`](../type-aliases/CreateDurableTaskWorkflowOpts.md)\<`I`, `O`\>

Durable task configuration options

##### Returns

[`TaskWorkflowDeclaration`](TaskWorkflowDeclaration.md)\<`I`, `O`\>

A TaskWorkflowDeclaration instance for a durable task

#### Call Signature

> **durableTask**\<`Fn`, `I`, `O`\>(`options`): [`TaskWorkflowDeclaration`](TaskWorkflowDeclaration.md)\<`I`, `O`\>

Defined in: [src/v1/client/client.ts:197](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/client.ts#L197)

Creates a new durable task workflow with types inferred from the function parameter.

##### Type Parameters

###### Fn

`Fn` *extends* (`input`, `ctx`) => `O` \| `Promise`\<`O`\>

The type of the durable task function with input and output extending JsonObject

###### I

`I` *extends* [`JsonObject`](../type-aliases/JsonObject.md) = `Parameters`\<`Fn`\>\[`0`\]

###### O

`O` *extends* [`OutputType`](../type-aliases/OutputType.md) = `ReturnType`\<`Fn`\> *extends* `Promise`\<`P`\> ? `P` *extends* [`OutputType`](../type-aliases/OutputType.md) ? `P`\<`P`\> : `void` : `ReturnType`\<`Fn`\> *extends* [`OutputType`](../type-aliases/OutputType.md) ? `ReturnType`\<`ReturnType`\<`Fn`\>\> : `void`

##### Parameters

###### options

`object` & `Omit`\<[`CreateDurableTaskWorkflowOpts`](../type-aliases/CreateDurableTaskWorkflowOpts.md)\<`I`, `O`\>, `"fn"`\>

Durable task configuration options with function that defines types

##### Returns

[`TaskWorkflowDeclaration`](TaskWorkflowDeclaration.md)\<`I`, `O`\>

A TaskWorkflowDeclaration instance with inferred types

***

### run()

> **run**\<`I`, `O`\>(`workflow`, `input`, `options`): `Promise`\<`O`\>

Defined in: [src/v1/client/client.ts:273](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/client.ts#L273)

Triggers a workflow run and waits for the result.

#### Type Parameters

##### I

`I` *extends* [`JsonObject`](../type-aliases/JsonObject.md) = [`UnknownInputType`](../type-aliases/UnknownInputType.md)

The input type for the workflow

##### O

`O` *extends* [`OutputType`](../type-aliases/OutputType.md) = `void`

The return type of the workflow

#### Parameters

##### workflow

The workflow to run, either as a Workflow instance or workflow name

`string` | [`Workflow`](../interfaces/Workflow.md) | [`BaseWorkflowDeclaration`](BaseWorkflowDeclaration.md)\<`I`, `O`\>

##### input

`I`

The input data for the workflow

##### options

[`RunOpts`](../type-aliases/RunOpts.md) = `{}`

Configuration options for the workflow run

#### Returns

`Promise`\<`O`\>

A promise that resolves with the workflow result

***

### runAndWait()

> **runAndWait**\<`I`, `O`\>(`workflow`, `input`, `options`): `Promise`\<`O`\>

Defined in: [src/v1/client/client.ts:256](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/client.ts#L256)

#### Type Parameters

##### I

`I` *extends* [`JsonObject`](../type-aliases/JsonObject.md) = [`UnknownInputType`](../type-aliases/UnknownInputType.md)

The input type for the workflow

##### O

`O` *extends* [`OutputType`](../type-aliases/OutputType.md) = `void`

The return type of the workflow

#### Parameters

##### workflow

The workflow to run, either as a Workflow instance or workflow name

`string` | [`Workflow`](../interfaces/Workflow.md) | [`BaseWorkflowDeclaration`](BaseWorkflowDeclaration.md)\<`I`, `O`\>

##### input

`I`

The input data for the workflow

##### options

[`RunOpts`](../type-aliases/RunOpts.md) = `{}`

Configuration options for the workflow run

#### Returns

`Promise`\<`O`\>

A promise that resolves with the workflow result

#### Alias

run
Triggers a workflow run and waits for the result.

***

### runNoWait()

> **runNoWait**\<`I`, `O`\>(`workflow`, `input`, `options`): `WorkflowRunRef`\<`O`\>

Defined in: [src/v1/client/client.ts:229](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/client.ts#L229)

Triggers a workflow run without waiting for completion.

#### Type Parameters

##### I

`I` *extends* [`JsonObject`](../type-aliases/JsonObject.md) = [`UnknownInputType`](../type-aliases/UnknownInputType.md)

The input type for the workflow

##### O

`O` *extends* [`OutputType`](../type-aliases/OutputType.md) = `void`

The return type of the workflow

#### Parameters

##### workflow

The workflow to run, either as a Workflow instance or workflow name

`string` | [`Workflow`](../interfaces/Workflow.md) | [`BaseWorkflowDeclaration`](BaseWorkflowDeclaration.md)\<`I`, `O`\>

##### input

`I`

The input data for the workflow

##### options

[`RunOpts`](../type-aliases/RunOpts.md)

Configuration options for the workflow run

#### Returns

`WorkflowRunRef`\<`O`\>

A WorkflowRunRef containing the run ID and methods to interact with the run

***

### runRef()

> **runRef**\<`T`\>(`id`): `WorkflowRunRef`\<`T`\>

Defined in: [src/v1/client/client.ts:447](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/client.ts#L447)

#### Type Parameters

##### T

`T` *extends* `Record`\<`string`, `any`\> = `any`

#### Parameters

##### id

`string`

#### Returns

`WorkflowRunRef`\<`T`\>

***

### task()

Implementation of the task method.

#### Call Signature

> **task**\<`I`, `O`\>(`options`): [`TaskWorkflowDeclaration`](TaskWorkflowDeclaration.md)\<`I`, `O`\>

Defined in: [src/v1/client/client.ts:146](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/client.ts#L146)

Creates a new task workflow.
Types can be explicitly specified as generics or inferred from the function signature.

##### Type Parameters

###### I

`I` *extends* [`JsonObject`](../type-aliases/JsonObject.md) = [`UnknownInputType`](../type-aliases/UnknownInputType.md)

The input type for the task

###### O

`O` *extends* [`OutputType`](../type-aliases/OutputType.md) = `void`

The output type of the task

##### Parameters

###### options

[`CreateTaskWorkflowOpts`](../type-aliases/CreateTaskWorkflowOpts.md)\<`I`, `O`\>

Task configuration options

##### Returns

[`TaskWorkflowDeclaration`](TaskWorkflowDeclaration.md)\<`I`, `O`\>

A TaskWorkflowDeclaration instance

#### Call Signature

> **task**\<`Fn`, `I`, `O`\>(`options`): [`TaskWorkflowDeclaration`](TaskWorkflowDeclaration.md)\<`I`, `O`\>

Defined in: [src/v1/client/client.ts:156](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/client.ts#L156)

Creates a new task workflow with types inferred from the function parameter.

##### Type Parameters

###### Fn

`Fn` *extends* (`input`, `ctx?`) => `O` \| `Promise`\<`O`\>

The type of the task function with input and output extending JsonObject

###### I

`I` *extends* [`JsonObject`](../type-aliases/JsonObject.md) = [`UnknownInputType`](../type-aliases/UnknownInputType.md) \| `Parameters`\<`Fn`\>\[`0`\]

###### O

`O` *extends* [`OutputType`](../type-aliases/OutputType.md) = `ReturnType`\<`Fn`\> *extends* `Promise`\<`P`\> ? `P` *extends* [`OutputType`](../type-aliases/OutputType.md) ? `P`\<`P`\> : `void` : `ReturnType`\<`Fn`\> *extends* [`OutputType`](../type-aliases/OutputType.md) ? `ReturnType`\<`ReturnType`\<`Fn`\>\> : `void`

##### Parameters

###### options

`object` & `Omit`\<[`CreateTaskWorkflowOpts`](../type-aliases/CreateTaskWorkflowOpts.md)\<`I`, `O`\>, `"fn"`\>

Task configuration options with function that defines types

##### Returns

[`TaskWorkflowDeclaration`](TaskWorkflowDeclaration.md)\<`I`, `O`\>

A TaskWorkflowDeclaration instance with inferred types

***

### webhooks()

> **webhooks**(`workflows`): `WebhookHandler`

Defined in: [src/v1/client/client.ts:443](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/client.ts#L443)

Register a webhook with the worker

#### Parameters

##### workflows

[`Workflow`](../interfaces/Workflow.md)[]

The workflows to register on the webhooks

#### Returns

`WebhookHandler`

A promise that resolves when the webhook is registered

***

### worker()

> **worker**(`name`, `options?`): `Promise`\<[`Worker`](Worker.md)\>

Defined in: [src/v1/client/client.ts:427](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/client.ts#L427)

Creates a new worker instance for processing workflow tasks.

#### Parameters

##### name

`string`

##### options?

Configuration options for creating the worker

`number` | [`CreateWorkerOpts`](../interfaces/CreateWorkerOpts.md)

#### Returns

`Promise`\<[`Worker`](Worker.md)\>

A promise that resolves with a new HatchetWorker instance

***

### workflow()

> **workflow**\<`I`, `O`\>(`options`): [`WorkflowDeclaration`](WorkflowDeclaration.md)\<`I`, `O`\>

Defined in: [src/v1/client/client.ts:132](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/client.ts#L132)

Creates a new workflow definition.

#### Type Parameters

##### I

`I` *extends* [`JsonObject`](../type-aliases/JsonObject.md) = [`UnknownInputType`](../type-aliases/UnknownInputType.md)

The input type for the workflow

##### O

`O` *extends* `void` \| `Record`\<`string`, [`JsonObject`](../type-aliases/JsonObject.md)\> & `object` = \{ \}

The return type of the workflow

#### Parameters

##### options

[`CreateWorkflowOpts`](../type-aliases/CreateWorkflowOpts.md)

Configuration options for creating the workflow

#### Returns

[`WorkflowDeclaration`](WorkflowDeclaration.md)\<`I`, `O`\>

A new Workflow instance

#### Note

It is possible to create an orphaned workflow if no client is available using @hatchet/client CreateWorkflow

***

### init()

> `static` **init**(`config?`, `options?`, `axiosConfig?`): `HatchetClient`

Defined in: [src/v1/client/client.ts:116](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/client.ts#L116)

Static factory method to create a new Hatchet client instance.

#### Parameters

##### config?

`Partial`\<`ClientConfig`\>

Optional configuration for the client

##### options?

`HatchetClientOptions`

Optional client options

##### axiosConfig?

`AxiosRequestConfig`\<`any`\>

Optional Axios configuration for HTTP requests

#### Returns

`HatchetClient`

A new Hatchet client instance
