[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / CreateWorkerOpts

# Interface: CreateWorkerOpts

Defined in: [src/v1/client/worker.ts:16](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/worker.ts#L16)

Options for creating a new hatchet worker
 CreateWorkerOpts

## Properties

### durableSlots?

> `optional` **durableSlots**: `number`

Defined in: [src/v1/client/worker.ts:29](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/worker.ts#L29)

(optional) Maximum number of concurrent runs on the durable worker, defaults to 1,000

***

### handleKill?

> `optional` **handleKill**: `boolean`

Defined in: [src/v1/client/worker.ts:24](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/worker.ts#L24)

(optional) Whether to handle kill signals

***

### labels?

> `optional` **labels**: `WorkerLabels`

Defined in: [src/v1/client/worker.ts:22](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/worker.ts#L22)

(optional) Worker labels for affinity-based assignment

***

### ~~maxRuns?~~

> `optional` **maxRuns**: `number`

Defined in: [src/v1/client/worker.ts:26](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/worker.ts#L26)

#### Deprecated

Use slots instead

***

### slots?

> `optional` **slots**: `number`

Defined in: [src/v1/client/worker.ts:18](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/worker.ts#L18)

(optional) Maximum number of concurrent runs on this worker, defaults to 100

***

### workflows?

> `optional` **workflows**: [`Workflow`](Workflow.md)[] \| [`BaseWorkflowDeclaration`](../classes/BaseWorkflowDeclaration.md)\<`any`, `any`\>[]

Defined in: [src/v1/client/worker.ts:20](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/worker.ts#L20)

(optional) Array of workflows to register
