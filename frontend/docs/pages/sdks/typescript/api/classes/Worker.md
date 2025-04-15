[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / Worker

# Class: Worker

Defined in: [src/v1/client/worker.ts:35](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/worker.ts#L35)

HatchetWorker class for workflow execution runtime

## Constructors

### Constructor

> **new Worker**(`v1`, `v0`, `nonDurable`, `config`, `name`): `Worker`

Defined in: [src/v1/client/worker.ts:49](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/worker.ts#L49)

Creates a new HatchetWorker instance

#### Parameters

##### v1

[`HatchetClient`](HatchetClient.md)

##### v0

`InternalHatchetClient`

##### nonDurable

[`V0Worker`](V0Worker.md)

The V0 worker implementation

##### config

[`CreateWorkerOpts`](../interfaces/CreateWorkerOpts.md)

##### name

`string`

#### Returns

`Worker`

## Properties

### \_v0

> **\_v0**: `InternalHatchetClient`

Defined in: [src/v1/client/worker.ts:39](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/worker.ts#L39)

***

### \_v1

> **\_v1**: [`HatchetClient`](HatchetClient.md)

Defined in: [src/v1/client/worker.ts:38](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/worker.ts#L38)

***

### config

> **config**: [`CreateWorkerOpts`](../interfaces/CreateWorkerOpts.md)

Defined in: [src/v1/client/worker.ts:36](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/worker.ts#L36)

***

### durable?

> `optional` **durable**: [`V0Worker`](V0Worker.md)

Defined in: [src/v1/client/worker.ts:43](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/worker.ts#L43)

***

### name

> **name**: `string`

Defined in: [src/v1/client/worker.ts:37](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/worker.ts#L37)

***

### nonDurable

> **nonDurable**: [`V0Worker`](V0Worker.md)

Defined in: [src/v1/client/worker.ts:42](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/worker.ts#L42)

Internal reference to the underlying V0 worker implementation

## Methods

### getLabels()

> **getLabels**(): `WorkerLabels`

Defined in: [src/v1/client/worker.ts:166](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/worker.ts#L166)

Get the labels for the worker

#### Returns

`WorkerLabels`

The labels for the worker

***

### isPaused()

> **isPaused**(): `Promise`\<`boolean`\>

Defined in: [src/v1/client/worker.ts:179](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/worker.ts#L179)

#### Returns

`Promise`\<`boolean`\>

***

### pause()

> **pause**(): `Promise`\<`any`[]\>

Defined in: [src/v1/client/worker.ts:194](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/worker.ts#L194)

#### Returns

`Promise`\<`any`[]\>

***

### registerWebhook()

> **registerWebhook**(`webhook`): `Promise`\<`AxiosResponse`\<[`WebhookWorkerCreated`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WebhookWorkerCreated.md), `any`\>\>

Defined in: [src/v1/client/worker.ts:175](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/worker.ts#L175)

Register a webhook with the worker

#### Parameters

##### webhook

[`WebhookWorkerCreateRequest`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WebhookWorkerCreateRequest.md)

The webhook to register

#### Returns

`Promise`\<`AxiosResponse`\<[`WebhookWorkerCreated`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WebhookWorkerCreated.md), `any`\>\>

A promise that resolves when the webhook is registered

***

### ~~registerWorkflow()~~

> **registerWorkflow**(`workflow`): `Promise`\<`void`[]\>

Defined in: [src/v1/client/worker.ts:121](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/worker.ts#L121)

Registers a single workflow with the worker

#### Parameters

##### workflow

The workflow to register

[`Workflow`](../interfaces/Workflow.md) | [`BaseWorkflowDeclaration`](BaseWorkflowDeclaration.md)\<`any`, `any`\>

#### Returns

`Promise`\<`void`[]\>

A promise that resolves when the workflow is registered

#### Deprecated

use registerWorkflows instead

***

### registerWorkflows()

> **registerWorkflows**(`workflows?`): `Promise`\<`void`[]\>

Defined in: [src/v1/client/worker.ts:89](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/worker.ts#L89)

Registers workflows with the worker

#### Parameters

##### workflows?

([`Workflow`](../interfaces/Workflow.md) \| [`BaseWorkflowDeclaration`](BaseWorkflowDeclaration.md)\<`any`, `any`\>)[]

Array of workflows to register

#### Returns

`Promise`\<`void`[]\>

Array of registered workflow promises

***

### start()

> **start**(): `Promise`\<`void`[]\>

Defined in: [src/v1/client/worker.ts:129](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/worker.ts#L129)

Starts the worker

#### Returns

`Promise`\<`void`[]\>

Promise that resolves when the worker is stopped or killed

***

### stop()

> **stop**(): `Promise`\<`void`[]\>

Defined in: [src/v1/client/worker.ts:143](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/worker.ts#L143)

Stops the worker

#### Returns

`Promise`\<`void`[]\>

Promise that resolves when the worker stops

***

### unpause()

> **unpause**(): `Promise`\<`any`[]\>

Defined in: [src/v1/client/worker.ts:205](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/worker.ts#L205)

#### Returns

`Promise`\<`any`[]\>

***

### upsertLabels()

> **upsertLabels**(`labels`): `Promise`\<`WorkerLabels`\>

Defined in: [src/v1/client/worker.ts:158](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/worker.ts#L158)

Updates or inserts worker labels

#### Parameters

##### labels

`WorkerLabels`

Worker labels to update

#### Returns

`Promise`\<`WorkerLabels`\>

Promise that resolves when labels are updated

***

### create()

> `static` **create**(`v1`, `v0`, `name`, `options`): `Promise`\<`Worker`\>

Defined in: [src/v1/client/worker.ts:69](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/worker.ts#L69)

Creates and initializes a new HatchetWorker

#### Parameters

##### v1

[`HatchetClient`](HatchetClient.md)

##### v0

`InternalHatchetClient`

The HatchetClient instance

##### name

`string`

##### options

[`CreateWorkerOpts`](../interfaces/CreateWorkerOpts.md)

Worker creation options

#### Returns

`Promise`\<`Worker`\>

A new HatchetWorker instance
