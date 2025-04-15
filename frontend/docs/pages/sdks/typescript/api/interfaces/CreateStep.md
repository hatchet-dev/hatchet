[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / CreateStep

# Interface: ~~CreateStep\<T, K\>~~

Defined in: [src/step.ts:680](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L680)

A step is a unit of work that can be run by a worker.
It is defined by a name, a function that returns the next step, and optional configuration.

## Deprecated

use hatchet.workflows.task factory instead

## Extends

- `TypeOf`\<*typeof* [`CreateStepSchema`](../variables/CreateStepSchema.md)\>

## Type Parameters

### T

`T`

### K

`K`

## Properties

### ~~backoff?~~

> `optional` **backoff**: `object`

Defined in: [src/step.ts:61](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L61)

#### ~~factor?~~

> `optional` **factor**: `number`

#### ~~maxSeconds?~~

> `optional` **maxSeconds**: `number`

#### Inherited from

`z.infer.backoff`

***

### ~~name~~

> **name**: `string`

Defined in: [src/step.ts:55](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L55)

#### Inherited from

`z.infer.name`

***

### ~~parents?~~

> `optional` **parents**: `string`[]

Defined in: [src/step.ts:56](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L56)

#### Inherited from

`z.infer.parents`

***

### ~~rate\_limits?~~

> `optional` **rate\_limits**: `object`[]

Defined in: [src/step.ts:59](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L59)

#### ~~duration?~~

> `optional` **duration**: [`SECOND`](../enumerations/RateLimitDuration.md#second) \| [`MINUTE`](../enumerations/RateLimitDuration.md#minute) \| [`HOUR`](../enumerations/RateLimitDuration.md#hour) \| [`DAY`](../enumerations/RateLimitDuration.md#day) \| [`WEEK`](../enumerations/RateLimitDuration.md#week) \| [`MONTH`](../enumerations/RateLimitDuration.md#month) \| [`YEAR`](../enumerations/RateLimitDuration.md#year) \| [`UNRECOGNIZED`](../enumerations/RateLimitDuration.md#unrecognized)

#### ~~dynamicKey?~~

> `optional` **dynamicKey**: `string`

#### ~~key?~~

> `optional` **key**: `string`

#### ~~limit?~~

> `optional` **limit**: `string` \| `number`

#### ~~staticKey?~~

> `optional` **staticKey**: `string`

#### ~~units~~

> **units**: `string` \| `number`

#### Inherited from

`z.infer.rate_limits`

***

### ~~retries?~~

> `optional` **retries**: `number`

Defined in: [src/step.ts:58](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L58)

#### Inherited from

`z.infer.retries`

***

### ~~run~~

> **run**: [`StepRunFunction`](../type-aliases/StepRunFunction.md)\<`T`, `K`\>

Defined in: [src/step.ts:681](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L681)

***

### ~~timeout?~~

> `optional` **timeout**: `string`

Defined in: [src/step.ts:57](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L57)

#### Inherited from

`z.infer.timeout`

***

### ~~worker\_labels?~~

> `optional` **worker\_labels**: `Record`\<`string`, `undefined` \| `string` \| `number` \| \{ `comparator`: `EQUAL` \| `NOT_EQUAL` \| `GREATER_THAN` \| `GREATER_THAN_OR_EQUAL` \| `LESS_THAN` \| `LESS_THAN_OR_EQUAL` \| `UNRECOGNIZED`; `required`: `boolean`; `value`: `string` \| `number`; `weight`: `number`; \}\>

Defined in: [src/step.ts:60](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L60)

#### Inherited from

`z.infer.worker_labels`
