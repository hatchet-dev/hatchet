[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / DurableContext

# Class: DurableContext\<T, K\>

Defined in: [src/step.ts:624](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L624)

## Extends

- [`Context`](Context.md)\<`T`, `K`\>

## Type Parameters

### T

`T`

### K

`K` = \{ \}

## Constructors

### Constructor

> **new DurableContext**\<`T`, `K`\>(`action`, `client`, `worker`): `DurableContext`\<`T`, `K`\>

Defined in: [src/step.ts:143](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L143)

#### Parameters

##### action

`Action`

##### client

`InternalHatchetClient`

##### worker

[`V0Worker`](V0Worker.md)

#### Returns

`DurableContext`\<`T`, `K`\>

#### Inherited from

[`Context`](Context.md).[`constructor`](Context.md#constructor)

## Properties

### action

> **action**: `Action`

Defined in: [src/step.ts:133](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L133)

#### Inherited from

[`Context`](Context.md).[`action`](Context.md#action)

***

### client

> **client**: `InternalHatchetClient`

Defined in: [src/step.ts:134](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L134)

#### Inherited from

[`Context`](Context.md).[`client`](Context.md#client)

***

### controller

> **controller**: `AbortController`

Defined in: [src/step.ts:132](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L132)

#### Inherited from

[`Context`](Context.md).[`controller`](Context.md#controller)

***

### data

> **data**: `ContextData`\<`T`, `K`\>

Defined in: [src/step.ts:128](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L128)

#### Inherited from

[`Context`](Context.md).[`data`](Context.md#data)

***

### input

> **input**: `T`

Defined in: [src/step.ts:130](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L130)

#### Inherited from

[`Context`](Context.md).[`input`](Context.md#input)

***

### logger

> **logger**: `Logger`

Defined in: [src/step.ts:139](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L139)

#### Inherited from

[`Context`](Context.md).[`logger`](Context.md#logger)

***

### overridesData

> **overridesData**: `Record`\<`string`, `any`\> = `{}`

Defined in: [src/step.ts:138](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L138)

#### Inherited from

[`Context`](Context.md).[`overridesData`](Context.md#overridesdata)

***

### spawnIndex

> **spawnIndex**: `number` = `0`

Defined in: [src/step.ts:141](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L141)

#### Inherited from

[`Context`](Context.md).[`spawnIndex`](Context.md#spawnindex)

***

### waitKey

> **waitKey**: `number` = `0`

Defined in: [src/step.ts:625](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L625)

***

### worker

> **worker**: [`ContextWorker`](ContextWorker.md)

Defined in: [src/step.ts:136](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L136)

#### Inherited from

[`Context`](Context.md).[`worker`](Context.md#worker)

## Accessors

### abortController

#### Get Signature

> **get** **abortController**(): `AbortController`

Defined in: [src/step.ts:165](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L165)

##### Returns

`AbortController`

#### Inherited from

[`Context`](Context.md).[`abortController`](Context.md#abortcontroller)

***

### cancelled

#### Get Signature

> **get** **cancelled**(): `boolean`

Defined in: [src/step.ts:169](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L169)

##### Returns

`boolean`

#### Inherited from

[`Context`](Context.md).[`cancelled`](Context.md#cancelled)

## Methods

### additionalMetadata()

> **additionalMetadata**(): `Record`\<`string`, `string`\>

Defined in: [src/step.ts:576](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L576)

Retrieves additional metadata associated with the current workflow run.

#### Returns

`Record`\<`string`, `string`\>

A record of metadata key-value pairs.

#### Inherited from

[`Context`](Context.md).[`additionalMetadata`](Context.md#additionalmetadata)

***

### bulkRunChildren()

> **bulkRunChildren**\<`Q`, `P`\>(`children`): `Promise`\<`P`[]\>

Defined in: [src/step.ts:396](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L396)

Runs multiple children workflows in parallel and waits for all results.

#### Type Parameters

##### Q

`Q` *extends* [`JsonObject`](../type-aliases/JsonObject.md) = `any`

##### P

`P` *extends* [`JsonObject`](../type-aliases/JsonObject.md) = `any`

#### Parameters

##### children

`object`[]

An array of objects containing the workflow name, input data, and options for each workflow.

#### Returns

`Promise`\<`P`[]\>

A list of results from the children workflows.

#### Inherited from

[`Context`](Context.md).[`bulkRunChildren`](Context.md#bulkrunchildren)

***

### bulkRunNoWaitChildren()

> **bulkRunNoWaitChildren**\<`Q`, `P`\>(`children`): `Promise`\<`WorkflowRunRef`\<`P`\>[]\>

Defined in: [src/step.ts:381](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L381)

Runs multiple children workflows in parallel without waiting for their results.

#### Type Parameters

##### Q

`Q` *extends* [`JsonObject`](../type-aliases/JsonObject.md) = `any`

##### P

`P` *extends* [`JsonObject`](../type-aliases/JsonObject.md) = `any`

#### Parameters

##### children

`object`[]

An array of  objects containing the workflow name, input data, and options for each workflow.

#### Returns

`Promise`\<`WorkflowRunRef`\<`P`\>[]\>

A list of workflow run references to the enqueued runs.

#### Inherited from

[`Context`](Context.md).[`bulkRunNoWaitChildren`](Context.md#bulkrunnowaitchildren)

***

### childIndex()

> **childIndex**(): `undefined` \| `number`

Defined in: [src/step.ts:590](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L590)

Gets the index of this workflow if it was spawned as part of a bulk operation.

#### Returns

`undefined` \| `number`

The child index number, or undefined if not set.

#### Inherited from

[`Context`](Context.md).[`childIndex`](Context.md#childindex)

***

### childKey()

> **childKey**(): `undefined` \| `string`

Defined in: [src/step.ts:598](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L598)

Gets the key associated with this workflow if it was spawned as a child workflow.

#### Returns

`undefined` \| `string`

The child key, or undefined if not set.

#### Inherited from

[`Context`](Context.md).[`childKey`](Context.md#childkey)

***

### errors()

> **errors**(): `Record`\<`string`, `string`\>

Defined in: [src/step.ts:220](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L220)

Returns errors from any task runs in the workflow.

#### Returns

`Record`\<`string`, `string`\>

A record mapping task names to error messages.

#### Throws

A warning if no errors are found (this method should be used in on-failure tasks).

#### Inherited from

[`Context`](Context.md).[`errors`](Context.md#errors)

***

### log()

> **log**(`message`, `level?`): `void`

Defined in: [src/step.ts:319](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L319)

Logs a message from the current task.

#### Parameters

##### message

`string`

The message to log.

##### level?

`LogLevel`

The log level (optional).

#### Returns

`void`

#### Inherited from

[`Context`](Context.md).[`log`](Context.md#log)

***

### parentOutput()

> **parentOutput**\<`L`\>(`parentTask`): `Promise`\<`L`\>

Defined in: [src/step.ts:179](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L179)

Retrieves the output of a parent task.

#### Type Parameters

##### L

`L` *extends* [`OutputType`](../type-aliases/OutputType.md)

#### Parameters

##### parentTask

The a CreateTaskOpts or string of the parent task name.

`string` | [`CreateWorkflowTaskOpts`](../type-aliases/CreateWorkflowTaskOpts.md)\<`any`, `L`\>

#### Returns

`Promise`\<`L`\>

The output of the specified parent task.

#### Throws

An error if the task output is not found.

#### Inherited from

[`Context`](Context.md).[`parentOutput`](Context.md#parentoutput)

***

### parentWorkflowRunId()

> **parentWorkflowRunId**(): `undefined` \| `string`

Defined in: [src/step.ts:606](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L606)

Gets the ID of the parent workflow run if this workflow was spawned as a child.

#### Returns

`undefined` \| `string`

The parent workflow run ID, or undefined if not a child workflow.

#### Inherited from

[`Context`](Context.md).[`parentWorkflowRunId`](Context.md#parentworkflowrunid)

***

### priority()

> **priority**(): `undefined` \| [`Priority`](../enumerations/Priority.md)

Defined in: [src/step.ts:610](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L610)

#### Returns

`undefined` \| [`Priority`](../enumerations/Priority.md)

#### Inherited from

[`Context`](Context.md).[`priority`](Context.md#priority)

***

### putStream()

> **putStream**(`data`): `Promise`\<`void`\>

Defined in: [src/step.ts:364](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L364)

Streams data from the current task run.

#### Parameters

##### data

The data to stream (string or binary).

`string` | `Uint8Array`\<`ArrayBufferLike`\>

#### Returns

`Promise`\<`void`\>

A promise that resolves when the data has been streamed.

#### Inherited from

[`Context`](Context.md).[`putStream`](Context.md#putstream)

***

### refreshTimeout()

> **refreshTimeout**(`incrementBy`): `Promise`\<`void`\>

Defined in: [src/step.ts:336](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L336)

Refreshes the timeout for the current task.

#### Parameters

##### incrementBy

[`Duration`](../type-aliases/Duration.md)

The interval by which to increment the timeout.
The interval should be specified in the format of '10s' for 10 seconds, '1m' for 1 minute, or '1d' for 1 day.

#### Returns

`Promise`\<`void`\>

#### Inherited from

[`Context`](Context.md).[`refreshTimeout`](Context.md#refreshtimeout)

***

### releaseSlot()

> **releaseSlot**(): `Promise`\<`void`\>

Defined in: [src/step.ts:353](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L353)

Releases a worker slot for a task run such that the worker can pick up another task.
Note: this is an advanced feature that may lead to unexpected behavior if used incorrectly.

#### Returns

`Promise`\<`void`\>

A promise that resolves when the slot has been released.

#### Inherited from

[`Context`](Context.md).[`releaseSlot`](Context.md#releaseslot)

***

### retryCount()

> **retryCount**(): `number`

Defined in: [src/step.ts:310](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L310)

Gets the number of times the current task has been retried.

#### Returns

`number`

The retry count.

#### Inherited from

[`Context`](Context.md).[`retryCount`](Context.md#retrycount)

***

### runChild()

> **runChild**\<`Q`, `P`\>(`workflow`, `input`, `options?`): `Promise`\<`P`\>

Defined in: [src/step.ts:491](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L491)

Runs a new workflow and waits for its result.

#### Type Parameters

##### Q

`Q` *extends* [`JsonObject`](../type-aliases/JsonObject.md)

##### P

`P` *extends* [`JsonObject`](../type-aliases/JsonObject.md)

#### Parameters

##### workflow

The workflow to run (name, Workflow instance, or WorkflowV1 instance).

`string` | [`Workflow`](../interfaces/Workflow.md) | [`BaseWorkflowDeclaration`](BaseWorkflowDeclaration.md)\<`Q`, `P`\> | [`TaskWorkflowDeclaration`](TaskWorkflowDeclaration.md)\<`Q`, `P`\>

##### input

`Q`

The input data for the workflow.

##### options?

`ChildRunOpts`

An options object containing key, sticky, priority, and additionalMetadata.

#### Returns

`Promise`\<`P`\>

The result of the workflow.

#### Inherited from

[`Context`](Context.md).[`runChild`](Context.md#runchild)

***

### runNoWaitChild()

> **runNoWaitChild**\<`Q`, `P`\>(`workflow`, `input`, `options?`): `WorkflowRunRef`\<`P`\>

Defined in: [src/step.ts:508](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L508)

Enqueues a new workflow without waiting for its result.

#### Type Parameters

##### Q

`Q` *extends* [`JsonObject`](../type-aliases/JsonObject.md)

##### P

`P` *extends* [`JsonObject`](../type-aliases/JsonObject.md)

#### Parameters

##### workflow

The workflow to enqueue (name, Workflow instance, or WorkflowV1 instance).

`string` | [`Workflow`](../interfaces/Workflow.md) | [`BaseWorkflowDeclaration`](BaseWorkflowDeclaration.md)\<`Q`, `P`\>

##### input

`Q`

The input data for the workflow.

##### options?

`ChildRunOpts`

An options object containing key, sticky, priority, and additionalMetadata.

#### Returns

`WorkflowRunRef`\<`P`\>

A reference to the spawned workflow run.

#### Inherited from

[`Context`](Context.md).[`runNoWaitChild`](Context.md#runnowaitchild)

***

### sleepFor()

> **sleepFor**(`duration`, `readableDataKey?`): `Promise`\<`Record`\<`string`, `any`\>\>

Defined in: [src/step.ts:633](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L633)

Pauses execution for the specified duration.
Duration is "global" meaning it will wait in real time regardless of transient failures like worker restarts.

#### Parameters

##### duration

[`Duration`](../type-aliases/Duration.md)

The duration to sleep for.

##### readableDataKey?

`string`

#### Returns

`Promise`\<`Record`\<`string`, `any`\>\>

A promise that resolves when the sleep duration has elapsed.

***

### ~~spawnWorkflow()~~

> **spawnWorkflow**\<`Q`, `P`\>(`workflow`, `input`, `options?`): `WorkflowRunRef`\<`P`\>

Defined in: [src/step.ts:525](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L525)

Spawns a new workflow.

#### Type Parameters

##### Q

`Q` *extends* [`JsonObject`](../type-aliases/JsonObject.md)

##### P

`P` *extends* [`JsonObject`](../type-aliases/JsonObject.md)

#### Parameters

##### workflow

The workflow to spawn (name, Workflow instance, or WorkflowV1 instance).

`string` | [`Workflow`](../interfaces/Workflow.md) | [`BaseWorkflowDeclaration`](BaseWorkflowDeclaration.md)\<`Q`, `P`\> | [`TaskWorkflowDeclaration`](TaskWorkflowDeclaration.md)\<`Q`, `P`\>

##### input

`Q`

The input data for the workflow.

##### options?

`ChildRunOpts`

Additional options for spawning the workflow.

#### Returns

`WorkflowRunRef`\<`P`\>

A reference to the spawned workflow run.

#### Deprecated

Use runChild or runNoWaitChild instead.

#### Inherited from

[`Context`](Context.md).[`spawnWorkflow`](Context.md#spawnworkflow)

***

### ~~spawnWorkflows()~~

> **spawnWorkflows**\<`Q`, `P`\>(`workflows`): `Promise`\<`WorkflowRunRef`\<`P`\>[]\>

Defined in: [src/step.ts:414](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L414)

Spawns multiple workflows.

#### Type Parameters

##### Q

`Q` *extends* [`JsonObject`](../type-aliases/JsonObject.md) = `any`

##### P

`P` *extends* [`JsonObject`](../type-aliases/JsonObject.md) = `any`

#### Parameters

##### workflows

`object`[]

An array of objects containing the workflow name, input data, and options for each workflow.

#### Returns

`Promise`\<`WorkflowRunRef`\<`P`\>[]\>

A list of references to the spawned workflow runs.

#### Deprecated

Use bulkRunNoWaitChildren or bulkRunChildren instead.

#### Inherited from

[`Context`](Context.md).[`spawnWorkflows`](Context.md#spawnworkflows)

***

### ~~stepName()~~

> **stepName**(): `string`

Defined in: [src/step.ts:278](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L278)

Gets the name of the current task.

#### Returns

`string`

The name of the task.

#### Deprecated

use ctx.taskName instead

#### Inherited from

[`Context`](Context.md).[`stepName`](Context.md#stepname)

***

### ~~stepOutput()~~

> **stepOutput**\<`L`\>(`step`): `L`

Defined in: [src/step.ts:195](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L195)

Get the output of a task.

#### Type Parameters

##### L

`L` = [`NextStep`](../type-aliases/NextStep.md)

#### Parameters

##### step

`string`

#### Returns

`L`

The output of the task.

#### Throws

An error if the task output is not found.

#### Deprecated

use ctx.parentOutput instead

#### Inherited from

[`Context`](Context.md).[`stepOutput`](Context.md#stepoutput)

***

### ~~stepRunErrors()~~

> **stepRunErrors**(): `Record`\<`string`, `string`\>

Defined in: [src/step.ts:211](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L211)

Returns errors from any task runs in the workflow.

#### Returns

`Record`\<`string`, `string`\>

A record mapping task names to error messages.

#### Throws

A warning if no errors are found (this method should be used in on-failure tasks).

#### Deprecated

use ctx.errors() instead

#### Inherited from

[`Context`](Context.md).[`stepRunErrors`](Context.md#steprunerrors)

***

### taskName()

> **taskName**(): `string`

Defined in: [src/step.ts:286](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L286)

Gets the name of the current running task.

#### Returns

`string`

The name of the task.

#### Inherited from

[`Context`](Context.md).[`taskName`](Context.md#taskname)

***

### taskRunId()

> **taskRunId**(): `string`

Defined in: [src/step.ts:302](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L302)

Gets the ID of the current task run.

#### Returns

`string`

The task run ID.

#### Inherited from

[`Context`](Context.md).[`taskRunId`](Context.md#taskrunid)

***

### triggeredByEvent()

> **triggeredByEvent**(): `boolean`

Defined in: [src/step.ts:244](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L244)

Determines if the workflow was triggered by an event.

#### Returns

`boolean`

True if the workflow was triggered by an event, otherwise false.

#### Inherited from

[`Context`](Context.md).[`triggeredByEvent`](Context.md#triggeredbyevent)

***

### triggers()

> **triggers**(): `TriggerData`

Defined in: [src/step.ts:236](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L236)

Gets the dag conditional triggers for the current workflow run.

#### Returns

`TriggerData`

The triggers for the current workflow.

#### Inherited from

[`Context`](Context.md).[`triggers`](Context.md#triggers)

***

### userData()

> **userData**(): `K`

Defined in: [src/step.ts:269](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L269)

Gets the user data associated with the workflow.

#### Returns

`K`

The user data.

#### Inherited from

[`Context`](Context.md).[`userData`](Context.md#userdata)

***

### waitFor()

> **waitFor**(`conditions`): `Promise`\<`Record`\<`string`, `any`\>\>

Defined in: [src/step.ts:643](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L643)

Pauses execution until the specified conditions are met.
Conditions are "global" meaning they will wait in real time regardless of transient failures like worker restarts.

#### Parameters

##### conditions

The conditions to wait for.

[`Conditions`](../type-aliases/Conditions.md) | [`Conditions`](../type-aliases/Conditions.md)[]

#### Returns

`Promise`\<`Record`\<`string`, `any`\>\>

A promise that resolves with the event that satisfied the conditions.

***

### ~~workflowInput()~~

> **workflowInput**(): `T`

Defined in: [src/step.ts:253](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L253)

Gets the input data for the current workflow.

#### Returns

`T`

The input data for the workflow.

#### Deprecated

use task input parameter instead

#### Inherited from

[`Context`](Context.md).[`workflowInput`](Context.md#workflowinput)

***

### workflowName()

> **workflowName**(): `string`

Defined in: [src/step.ts:261](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L261)

Gets the name of the current workflow.

#### Returns

`string`

The name of the workflow.

#### Inherited from

[`Context`](Context.md).[`workflowName`](Context.md#workflowname)

***

### workflowRunId()

> **workflowRunId**(): `string`

Defined in: [src/step.ts:294](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L294)

Gets the ID of the current workflow run.

#### Returns

`string`

The workflow run ID.

#### Inherited from

[`Context`](Context.md).[`workflowRunId`](Context.md#workflowrunid)
