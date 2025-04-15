[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / ListRunsOpts

# Interface: ListRunsOpts

Defined in: [src/v1/client/features/runs.ts:24](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/runs.ts#L24)

## Extends

- [`RunFilter`](../type-aliases/RunFilter.md)

## Properties

### additionalMetadata?

> `optional` **additionalMetadata**: `Record`\<`string`, `string`\>

Defined in: [src/v1/client/features/runs.ts:11](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/runs.ts#L11)

#### Inherited from

[`RunFilter`](../type-aliases/RunFilter.md).[`additionalMetadata`](../type-aliases/RunFilter.md#additionalmetadata)

***

### limit?

> `optional` **limit**: `number`

Defined in: [src/v1/client/features/runs.ts:34](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/runs.ts#L34)

The number to limit by

#### Format

int64

***

### offset?

> `optional` **offset**: `number`

Defined in: [src/v1/client/features/runs.ts:29](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/runs.ts#L29)

The number to skip

#### Format

int64

***

### onlyTasks

> **onlyTasks**: `boolean`

Defined in: [src/v1/client/features/runs.ts:45](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/runs.ts#L45)

Whether to include DAGs or only to include tasks

***

### parentTaskExternalId?

> `optional` **parentTaskExternalId**: `string`

Defined in: [src/v1/client/features/runs.ts:52](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/runs.ts#L52)

The parent task external id to filter by

#### Format

uuid

#### Min Length

36

#### Max Length

36

***

### since?

> `optional` **since**: `Date`

Defined in: [src/v1/client/features/runs.ts:7](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/runs.ts#L7)

#### Inherited from

[`RunFilter`](../type-aliases/RunFilter.md).[`since`](../type-aliases/RunFilter.md#since)

***

### statuses?

> `optional` **statuses**: [`V1TaskStatus`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/enumerations/V1TaskStatus.md)[]

Defined in: [src/v1/client/features/runs.ts:9](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/runs.ts#L9)

#### Inherited from

[`RunFilter`](../type-aliases/RunFilter.md).[`statuses`](../type-aliases/RunFilter.md#statuses)

***

### until?

> `optional` **until**: `Date`

Defined in: [src/v1/client/features/runs.ts:8](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/runs.ts#L8)

#### Inherited from

[`RunFilter`](../type-aliases/RunFilter.md).[`until`](../type-aliases/RunFilter.md#until)

***

### workerId?

> `optional` **workerId**: `string`

Defined in: [src/v1/client/features/runs.ts:43](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/runs.ts#L43)

The worker id to filter by

#### Format

uuid

#### Min Length

36

#### Max Length

36

***

### workflowNames?

> `optional` **workflowNames**: `string`[]

Defined in: [src/v1/client/features/runs.ts:10](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/runs.ts#L10)

#### Inherited from

[`RunFilter`](../type-aliases/RunFilter.md).[`workflowNames`](../type-aliases/RunFilter.md#workflownames)
