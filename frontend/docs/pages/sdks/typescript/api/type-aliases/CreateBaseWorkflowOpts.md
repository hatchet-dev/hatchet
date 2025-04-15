[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / CreateBaseWorkflowOpts

# Type Alias: CreateBaseWorkflowOpts

> **CreateBaseWorkflowOpts** = `object`

Defined in: [src/v1/declaration.ts:73](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L73)

## Properties

### concurrency?

> `optional` **concurrency**: [`Concurrency`](Concurrency.md) \| [`Concurrency`](Concurrency.md)[]

Defined in: [src/v1/declaration.ts:110](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L110)

(optional) concurrency config for the workflow.

***

### defaultPriority?

> `optional` **defaultPriority**: [`Priority`](../enumerations/Priority.md)

Defined in: [src/v1/declaration.ts:116](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L116)

(optional) the priority for the workflow.
values: Priority.LOW, Priority.MEDIUM, Priority.HIGH (1, 2, or 3 )

***

### description?

> `optional` **description**: [`Workflow`](../interfaces/Workflow.md)\[`"description"`\]

Defined in: [src/v1/declaration.ts:81](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L81)

(optional) description of the workflow.

***

### name

> **name**: [`Workflow`](../interfaces/Workflow.md)\[`"id"`\]

Defined in: [src/v1/declaration.ts:77](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L77)

The name of the workflow.

***

### ~~on?~~

> `optional` **on**: [`Workflow`](../interfaces/Workflow.md)\[`"on"`\]

Defined in: [src/v1/declaration.ts:95](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L95)

(optional) on config for the workflow.

#### Deprecated

use onCrons and onEvents instead

***

### onCrons?

> `optional` **onCrons**: `string`[]

Defined in: [src/v1/declaration.ts:100](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L100)

(optional) cron config for the workflow.

***

### onEvents?

> `optional` **onEvents**: `string`[]

Defined in: [src/v1/declaration.ts:105](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L105)

(optional) event config for the workflow.

***

### sticky?

> `optional` **sticky**: [`Workflow`](../interfaces/Workflow.md)\[`"sticky"`\]

Defined in: [src/v1/declaration.ts:89](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L89)

(optional) sticky strategy for the workflow.

***

### version?

> `optional` **version**: [`Workflow`](../interfaces/Workflow.md)\[`"version"`\]

Defined in: [src/v1/declaration.ts:85](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L85)

(optional) version of the workflow.
