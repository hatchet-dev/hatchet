[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / BaseWorkflowDeclaration

# Class: BaseWorkflowDeclaration\<I, O\>

Defined in: [src/v1/declaration.ts:229](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L229)

Represents a workflow that can be executed by Hatchet.

## Extended by

- [`WorkflowDeclaration`](WorkflowDeclaration.md)
- [`TaskWorkflowDeclaration`](TaskWorkflowDeclaration.md)

## Type Parameters

### I

`I` *extends* [`InputType`](../type-aliases/InputType.md) = [`UnknownInputType`](../type-aliases/UnknownInputType.md)

The input type for the workflow.

### O

`O` *extends* [`OutputType`](../type-aliases/OutputType.md) = `void`

The return type of the workflow.

## Constructors

### Constructor

> **new BaseWorkflowDeclaration**\<`I`, `O`\>(`options`, `client?`): `BaseWorkflowDeclaration`\<`I`, `O`\>

Defined in: [src/v1/declaration.ts:248](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L248)

Creates a new workflow instance.

#### Parameters

##### options

[`CreateWorkflowOpts`](../type-aliases/CreateWorkflowOpts.md)

The options for creating the workflow.

##### client?

`IHatchetClient`

Optional Hatchet client instance.

#### Returns

`BaseWorkflowDeclaration`\<`I`, `O`\>

## Properties

### client

> **client**: `undefined` \| `IHatchetClient`

Defined in: [src/v1/declaration.ts:236](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L236)

The Hatchet client instance used to execute the workflow.

***

### definition

> **definition**: [`WorkflowDefinition`](../type-aliases/WorkflowDefinition.md)

Defined in: [src/v1/declaration.ts:241](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L241)

The internal workflow definition.

## Accessors

### id

#### Get Signature

> **get** **id**(): `string`

Defined in: [src/v1/declaration.ts:492](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L492)

##### Returns

`string`

***

### name

#### Get Signature

> **get** **name**(): `string`

Defined in: [src/v1/declaration.ts:500](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L500)

Get the friendly name of the workflow.

##### Returns

`string`

The name of the workflow.

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

***

### run()

#### Call Signature

> **run**(`input`, `options?`, `_standaloneTaskName?`): `Promise`\<`O`\>

Defined in: [src/v1/declaration.ts:312](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L312)

Executes the workflow with the given input and awaits the results.

##### Parameters

###### input

`I`

The input data for the workflow.

###### options?

[`RunOpts`](../type-aliases/RunOpts.md)

Optional configuration for this workflow run.

###### \_standaloneTaskName?

`string`

##### Returns

`Promise`\<`O`\>

A promise that resolves with the workflow result.

##### Throws

Error if the workflow is not bound to a Hatchet client.

#### Call Signature

> **run**(`input`, `options?`, `_standaloneTaskName?`): `Promise`\<`O`[]\>

Defined in: [src/v1/declaration.ts:313](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L313)

Executes the workflow with the given input and awaits the results.

##### Parameters

###### input

`I`[]

The input data for the workflow.

###### options?

[`RunOpts`](../type-aliases/RunOpts.md)

Optional configuration for this workflow run.

###### \_standaloneTaskName?

`string`

##### Returns

`Promise`\<`O`[]\>

A promise that resolves with the workflow result.

##### Throws

Error if the workflow is not bound to a Hatchet client.

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

***

### runNoWait()

> **runNoWait**(`input`, `options?`, `_standaloneTaskName?`): `WorkflowRunRef`\<`O`\>

Defined in: [src/v1/declaration.ts:265](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L265)

Triggers a workflow run without waiting for completion.

#### Parameters

##### input

`I`

The input data for the workflow.

##### options?

[`RunOpts`](../type-aliases/RunOpts.md)

Optional configuration for this workflow run.

##### \_standaloneTaskName?

`string`

#### Returns

`WorkflowRunRef`\<`O`\>

A WorkflowRunRef containing the run ID and methods to get results and interact with the run.

#### Throws

Error if the workflow is not bound to a Hatchet client.

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
