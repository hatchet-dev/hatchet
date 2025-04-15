[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / Workflow

# Interface: ~~Workflow~~

Defined in: [src/workflow.ts:59](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/workflow.ts#L59)

## Deprecated

Use client.workflow instead (TODO link to migration doc)

## Extends

- `TypeOf`\<*typeof* [`CreateWorkflowSchema`](../variables/CreateWorkflowSchema.md)\>

## Properties

### ~~concurrency?~~

> `optional` **concurrency**: `object` & `object`

Defined in: [src/workflow.ts:60](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/workflow.ts#L60)

#### Type declaration

##### ~~expression?~~

> `optional` **expression**: `string`

##### ~~limitStrategy?~~

> `optional` **limitStrategy**: `CANCEL_IN_PROGRESS` \| `DROP_NEWEST` \| `QUEUE_NEWEST` \| `GROUP_ROUND_ROBIN` \| `CANCEL_NEWEST` \| `UNRECOGNIZED`

##### ~~maxRuns?~~

> `optional` **maxRuns**: `number`

##### ~~name~~

> **name**: `string`

#### Type declaration

##### ~~key()?~~

> `optional` **key**: (`ctx`) => `string`

###### Parameters

###### ctx

`any`

###### Returns

`string`

***

### ~~description~~

> **description**: `string`

Defined in: [src/workflow.ts:40](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/workflow.ts#L40)

#### Inherited from

`z.infer.description`

***

### ~~id~~

> **id**: `string`

Defined in: [src/workflow.ts:39](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/workflow.ts#L39)

#### Inherited from

`z.infer.id`

***

### ~~on?~~

> `optional` **on**: \{ `cron`: `string`; `event`: `undefined`; \} \| \{ `cron`: `undefined`; `event`: `string`; \} = `OnConfigSchema`

Defined in: [src/workflow.ts:51](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/workflow.ts#L51)

#### Inherited from

`z.infer.on`

***

### ~~onFailure?~~

> `optional` **onFailure**: [`CreateStep`](CreateStep.md)\<`any`, `any`\>

Defined in: [src/workflow.ts:64](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/workflow.ts#L64)

#### Overrides

`z.infer.onFailure`

***

### ~~scheduleTimeout?~~

> `optional` **scheduleTimeout**: `string`

Defined in: [src/workflow.ts:46](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/workflow.ts#L46)

#### Inherited from

`z.infer.scheduleTimeout`

***

### ~~steps~~

> **steps**: [`CreateStep`](CreateStep.md)\<`any`, `any`\>[]

Defined in: [src/workflow.ts:63](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/workflow.ts#L63)

#### Overrides

`z.infer.steps`

***

### ~~sticky?~~

> `optional` **sticky**: `SOFT` \| `HARD` \| `UNRECOGNIZED`

Defined in: [src/workflow.ts:45](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/workflow.ts#L45)

sticky will attempt to run all steps for workflow on the same worker

#### Inherited from

`z.infer.sticky`

***

### ~~timeout?~~

> `optional` **timeout**: `string`

Defined in: [src/workflow.ts:50](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/workflow.ts#L50)

#### Deprecated

Workflow timeout is deprecated. Use step timeouts instead.

#### Inherited from

`z.infer.timeout`

***

### ~~version?~~

> `optional` **version**: `string`

Defined in: [src/workflow.ts:41](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/workflow.ts#L41)

#### Inherited from

`z.infer.version`
