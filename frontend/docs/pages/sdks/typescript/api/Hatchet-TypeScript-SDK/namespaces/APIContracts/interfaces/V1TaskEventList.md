[**Hatchet TypeScript SDK**](../../../../README.md)

***

[Hatchet TypeScript SDK](../../../../README.md) / [APIContracts](../README.md) / V1TaskEventList

# Interface: V1TaskEventList

Defined in: [src/clients/rest/generated/data-contracts.ts:1568](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1568)

## Properties

### pagination?

> `optional` **pagination**: [`PaginationResponse`](PaginationResponse.md)

Defined in: [src/clients/rest/generated/data-contracts.ts:1569](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1569)

***

### rows?

> `optional` **rows**: `object`[]

Defined in: [src/clients/rest/generated/data-contracts.ts:1570](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1570)

#### errorMessage?

> `optional` **errorMessage**: `string`

#### eventType

> **eventType**: `"REQUEUED_NO_WORKER"` \| `"REQUEUED_RATE_LIMIT"` \| `"SCHEDULING_TIMED_OUT"` \| `"ASSIGNED"` \| `"STARTED"` \| `"FINISHED"` \| `"FAILED"` \| `"RETRYING"` \| `"CANCELLED"` \| `"TIMED_OUT"` \| `"REASSIGNED"` \| `"SLOT_RELEASED"` \| `"TIMEOUT_REFRESHED"` \| `"RETRIED_BY_USER"` \| `"SENT_TO_WORKER"` \| `"RATE_LIMIT_ERROR"` \| `"ACKNOWLEDGED"` \| `"CREATED"` \| `"QUEUED"` \| `"SKIPPED"`

#### id

> **id**: `number`

#### message

> **message**: `string`

#### output?

> `optional` **output**: `string`

#### taskDisplayName?

> `optional` **taskDisplayName**: `string`

#### taskId

> **taskId**: `string`

##### Format

uuid

#### timestamp

> **timestamp**: `string`

##### Format

date-time

#### workerId?

> `optional` **workerId**: `string`

##### Format

uuid
