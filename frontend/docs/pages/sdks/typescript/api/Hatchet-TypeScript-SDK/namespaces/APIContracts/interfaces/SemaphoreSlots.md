[**Hatchet TypeScript SDK**](../../../../README.md)

***

[Hatchet TypeScript SDK](../../../../README.md) / [APIContracts](../README.md) / SemaphoreSlots

# Interface: SemaphoreSlots

Defined in: [src/clients/rest/generated/data-contracts.ts:1135](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1135)

## Properties

### actionId

> **actionId**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1142](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1142)

The action id.

***

### startedAt?

> `optional` **startedAt**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1147](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1147)

The time this slot was started.

#### Format

date-time

***

### status

> **status**: [`StepRunStatus`](../enumerations/StepRunStatus.md)

Defined in: [src/clients/rest/generated/data-contracts.ts:1158](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1158)

***

### stepRunId

> **stepRunId**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1140](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1140)

The step run id.

#### Format

uuid

***

### timeoutAt?

> `optional` **timeoutAt**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1152](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1152)

The time this slot will timeout.

#### Format

date-time

***

### workflowRunId

> **workflowRunId**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1157](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1157)

The workflow run id.

#### Format

uuid
