[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / TaskWorkflowDeclaration

# Class: TaskWorkflowDeclaration\<I, O\>

Defined in: [src/v1/declaration.ts:659](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L659)

Represents a workflow that can be executed by Hatchet.

## Extends

- [`BaseWorkflowDeclaration`](BaseWorkflowDeclaration.md)\<`I`, `O`\>

## Type Parameters

### I

`I` *extends* [`InputType`](../type-aliases/InputType.md) = [`UnknownInputType`](../type-aliases/UnknownInputType.md)

The input type for the workflow.

### O

`O` *extends* [`OutputType`](../type-aliases/OutputType.md) = `void`

The return type of the workflow.

## Constructors

### Constructor

> **new TaskWorkflowDeclaration**\<`I`, `O`\>(`options`, `client?`): `TaskWorkflowDeclaration`\<`I`, `O`\>

Defined in: [src/v1/declaration.ts:665](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L665)

#### Parameters

##### options

[`CreateTaskWorkflowOpts`](../type-aliases/CreateTaskWorkflowOpts.md)\<`I`, `O`\>

##### client?

`IHatchetClient`

#### Returns

`TaskWorkflowDeclaration`\<`I`, `O`\>

#### Overrides

[`BaseWorkflowDeclaration`](BaseWorkflowDeclaration.md).[`constructor`](BaseWorkflowDeclaration.md#constructor)

## Properties

### \_standalone\_task\_name

> **\_standalone\_task\_name**: `string`

Defined in: [src/v1/declaration.ts:663](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L663)

***

### client

> **client**: `undefined` \| `IHatchetClient`

Defined in: [src/v1/declaration.ts:236](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L236)

The Hatchet client instance used to execute the workflow.

#### Inherited from

[`BaseWorkflowDeclaration`](BaseWorkflowDeclaration.md).[`client`](BaseWorkflowDeclaration.md#client)

***

### definition

> **definition**: [`WorkflowDefinition`](../type-aliases/WorkflowDefinition.md)

Defined in: [src/v1/declaration.ts:241](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L241)

The internal workflow definition.

#### Inherited from

[`BaseWorkflowDeclaration`](BaseWorkflowDeclaration.md).[`definition`](BaseWorkflowDeclaration.md#definition)

## Accessors

### id

#### Get Signature

> **get** **id**(): `string`

Defined in: [src/v1/declaration.ts:492](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L492)

##### Returns

`string`

#### Inherited from

[`BaseWorkflowDeclaration`](BaseWorkflowDeclaration.md).[`id`](BaseWorkflowDeclaration.md#id)

***

### name

#### Get Signature

> **get** **name**(): `string`

Defined in: [src/v1/declaration.ts:500](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L500)

Get the friendly name of the workflow.

##### Returns

`string`

The name of the workflow.

#### Inherited from

[`BaseWorkflowDeclaration`](BaseWorkflowDeclaration.md).[`name`](BaseWorkflowDeclaration.md#name)

***

### taskDef

#### Get Signature

> **get** **taskDef**(): [`CreateWorkflowTaskOpts`](../type-aliases/CreateWorkflowTaskOpts.md)\<`any`, `any`\>

Defined in: [src/v1/declaration.ts:696](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L696)

##### Returns

[`CreateWorkflowTaskOpts`](../type-aliases/CreateWorkflowTaskOpts.md)\<`any`, `any`\>

## Methods

### cron()

> **cron**(`name`, `expression`, `input`, `options?`): `Promise`\<[`CronWorkflows`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/CronWorkflows.md)\>

Defined in: [src/v1/declaration.ts:399](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L399)

Creates a cron schedule for the workflow.

#### Parameters

##### name

`string`

The name of the cron schedule.

##### expression

`string`

The cron expression defining the schedule.

##### input

`I`

The input data for the workflow.

##### options?

[`RunOpts`](../type-aliases/RunOpts.md)

Optional configuration for this workflow run.

#### Returns

`Promise`\<[`CronWorkflows`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/CronWorkflows.md)\>

A promise that resolves with the cron workflow details.

#### Throws

Error if the workflow is not bound to a Hatchet client.

#### Inherited from

[`BaseWorkflowDeclaration`](BaseWorkflowDeclaration.md).[`cron`](BaseWorkflowDeclaration.md#cron)

***

### delay()

> **delay**(`duration`, `input`, `options?`): `Promise`\<[`ScheduledWorkflows`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/ScheduledWorkflows.md)\>

Defined in: [src/v1/declaration.ts:384](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L384)

Schedules a workflow to run after a specified delay.

#### Parameters

##### duration

`number`

The delay in seconds before the workflow should run.

##### input

`I`

The input data for the workflow.

##### options?

[`RunOpts`](../type-aliases/RunOpts.md)

Optional configuration for this workflow run.

#### Returns

`Promise`\<[`ScheduledWorkflows`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/ScheduledWorkflows.md)\>

A promise that resolves with the scheduled workflow details.

#### Throws

Error if the workflow is not bound to a Hatchet client.

#### Inherited from

[`BaseWorkflowDeclaration`](BaseWorkflowDeclaration.md).[`delay`](BaseWorkflowDeclaration.md#delay)

***

### get()

> **get**(): `Promise`\<`any`\>

Defined in: [src/v1/declaration.ts:456](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L456)

Get the current state of the workflow.

#### Returns

`Promise`\<`any`\>

A promise that resolves with the workflow state.

#### Throws

Error if the workflow is not bound to a Hatchet client.

#### Inherited from

[`BaseWorkflowDeclaration`](BaseWorkflowDeclaration.md).[`get`](BaseWorkflowDeclaration.md#get)

***

### metrics()

> **metrics**(`opts?`): `Promise`\<[`WorkflowMetrics`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowMetrics.md)\>

Defined in: [src/v1/declaration.ts:426](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L426)

Get metrics for the workflow.

#### Parameters

##### opts?

Optional configuration for the metrics request.

###### groupKey?

`string`

A group key to filter metrics by

###### status?

[`WorkflowRunStatus`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/enumerations/WorkflowRunStatus.md)

A status of workflow run statuses to filter by

#### Returns

`Promise`\<[`WorkflowMetrics`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowMetrics.md)\>

A promise that resolves with the workflow metrics.

#### Throws

Error if the workflow is not bound to a Hatchet client.

#### Inherited from

[`BaseWorkflowDeclaration`](BaseWorkflowDeclaration.md).[`metrics`](BaseWorkflowDeclaration.md#metrics)

***

### queueMetrics()

> **queueMetrics**(`opts?`): `Promise`\<[`TenantQueueMetrics`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/TenantQueueMetrics.md)\>

Defined in: [src/v1/declaration.ts:440](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L440)

Get queue metrics for the workflow.

#### Parameters

##### opts?

`Omit`\<`undefined` \| \{ `additionalMetadata`: `string`[]; `workflows`: `string`[]; \}, `"workflows"`\>

Optional configuration for the metrics request.

#### Returns

`Promise`\<[`TenantQueueMetrics`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/TenantQueueMetrics.md)\>

A promise that resolves with the workflow metrics.

#### Throws

Error if the workflow is not bound to a Hatchet client.

#### Inherited from

[`BaseWorkflowDeclaration`](BaseWorkflowDeclaration.md).[`queueMetrics`](BaseWorkflowDeclaration.md#queuemetrics)

***

### run()

#### Call Signature

> **run**(`input`, `options?`): `Promise`\<`O`\>

Defined in: [src/v1/declaration.ts:675](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L675)

Executes the workflow with the given input and awaits the results.

##### Parameters

###### input

`I`

The input data for the workflow.

###### options?

[`RunOpts`](../type-aliases/RunOpts.md)

Optional configuration for this workflow run.

##### Returns

`Promise`\<`O`\>

A promise that resolves with the workflow result.

##### Throws

Error if the workflow is not bound to a Hatchet client.

##### Overrides

[`BaseWorkflowDeclaration`](BaseWorkflowDeclaration.md).[`run`](BaseWorkflowDeclaration.md#run)

#### Call Signature

> **run**(`input`, `options?`): `Promise`\<`O`[]\>

Defined in: [src/v1/declaration.ts:676](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L676)

Executes the workflow with the given input and awaits the results.

##### Parameters

###### input

`I`[]

The input data for the workflow.

###### options?

[`RunOpts`](../type-aliases/RunOpts.md)

Optional configuration for this workflow run.

##### Returns

`Promise`\<`O`[]\>

A promise that resolves with the workflow result.

##### Throws

Error if the workflow is not bound to a Hatchet client.

##### Overrides

[`BaseWorkflowDeclaration`](BaseWorkflowDeclaration.md).[`run`](BaseWorkflowDeclaration.md#run)

***

### runAndWait()

#### Call Signature

> **runAndWait**(`input`, `options?`, `_standaloneTaskName?`): `Promise`\<`O`\>

Defined in: [src/v1/declaration.ts:288](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L288)

##### Parameters

###### input

`I`

The input data for the workflow

###### options?

[`RunOpts`](../type-aliases/RunOpts.md)

Configuration options for the workflow run

###### \_standaloneTaskName?

`string`

##### Returns

`Promise`\<`O`\>

A promise that resolves with the workflow result

##### Alias

run
Triggers a workflow run and waits for the result.

##### Inherited from

[`BaseWorkflowDeclaration`](BaseWorkflowDeclaration.md).[`runAndWait`](BaseWorkflowDeclaration.md#runandwait)

#### Call Signature

> **runAndWait**(`input`, `options?`, `_standaloneTaskName?`): `Promise`\<`O`[]\>

Defined in: [src/v1/declaration.ts:289](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L289)

##### Parameters

###### input

`I`[]

The input data for the workflow

###### options?

[`RunOpts`](../type-aliases/RunOpts.md)

Configuration options for the workflow run

###### \_standaloneTaskName?

`string`

##### Returns

`Promise`\<`O`[]\>

A promise that resolves with the workflow result

##### Alias

run
Triggers a workflow run and waits for the result.

##### Inherited from

[`BaseWorkflowDeclaration`](BaseWorkflowDeclaration.md).[`runAndWait`](BaseWorkflowDeclaration.md#runandwait)

***

### runNoWait()

> **runNoWait**(`input`, `options?`): `WorkflowRunRef`\<`O`\>

Defined in: [src/v1/declaration.ts:688](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L688)

Triggers a workflow run without waiting for completion.

#### Parameters

##### input

`I`

The input data for the workflow.

##### options?

[`RunOpts`](../type-aliases/RunOpts.md)

Optional configuration for this workflow run.

#### Returns

`WorkflowRunRef`\<`O`\>

A WorkflowRunRef containing the run ID and methods to get results and interact with the run.

#### Throws

Error if the workflow is not bound to a Hatchet client.

#### Overrides

[`BaseWorkflowDeclaration`](BaseWorkflowDeclaration.md).[`runNoWait`](BaseWorkflowDeclaration.md#runnowait)

***

### schedule()

> **schedule**(`enqueueAt`, `input`, `options?`): `Promise`\<[`ScheduledWorkflows`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/ScheduledWorkflows.md)\>

Defined in: [src/v1/declaration.ts:362](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L362)

Schedules a workflow to run at a specific date and time in the future.

#### Parameters

##### enqueueAt

`Date`

The date when the workflow should be triggered.

##### input

`I`

The input data for the workflow.

##### options?

[`RunOpts`](../type-aliases/RunOpts.md)

Optional configuration for this workflow run.

#### Returns

`Promise`\<[`ScheduledWorkflows`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/ScheduledWorkflows.md)\>

A promise that resolves with the scheduled workflow details.

#### Throws

Error if the workflow is not bound to a Hatchet client.

#### Inherited from

[`BaseWorkflowDeclaration`](BaseWorkflowDeclaration.md).[`schedule`](BaseWorkflowDeclaration.md#schedule)
