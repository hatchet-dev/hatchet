[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / ContextWorker

# Class: ContextWorker

Defined in: [src/step.ts:84](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L84)

## Constructors

### Constructor

> **new ContextWorker**(`worker`): `ContextWorker`

Defined in: [src/step.ts:86](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L86)

#### Parameters

##### worker

[`V0Worker`](V0Worker.md)

#### Returns

`ContextWorker`

## Methods

### hasWorkflow()

> **hasWorkflow**(`workflowName`): `boolean`

Defined in: [src/step.ts:103](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L103)

Checks if the worker has a registered workflow.

#### Parameters

##### workflowName

`string`

The name of the workflow to check.

#### Returns

`boolean`

True if the workflow is registered, otherwise false.

***

### id()

> **id**(): `undefined` \| `string`

Defined in: [src/step.ts:94](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L94)

Gets the ID of the worker.

#### Returns

`undefined` \| `string`

The ID of the worker.

***

### labels()

> **labels**(): `WorkerLabels`

Defined in: [src/step.ts:113](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L113)

Gets the current state of the worker labels.

#### Returns

`WorkerLabels`

The labels of the worker.

***

### upsertLabels()

> **upsertLabels**(`labels`): `Promise`\<`WorkerLabels`\>

Defined in: [src/step.ts:122](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L122)

Upserts the a set of labels on the worker.

#### Parameters

##### labels

`WorkerLabels`

The labels to upsert.

#### Returns

`Promise`\<`WorkerLabels`\>

A promise that resolves when the labels have been upserted.
