[**Hatchet TypeScript SDK**](../../../../README.md)

***

[Hatchet TypeScript SDK](../../../../README.md) / [APIContracts](../README.md) / WorkflowConcurrency

# Interface: WorkflowConcurrency

Defined in: [src/clients/rest/generated/data-contracts.ts:733](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L733)

## Properties

### getConcurrencyGroup

> **getConcurrencyGroup**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:742](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L742)

An action which gets the concurrency group for the WorkflowRun.

***

### limitStrategy

> **limitStrategy**: `"CANCEL_IN_PROGRESS"` \| `"DROP_NEWEST"` \| `"QUEUE_NEWEST"` \| `"GROUP_ROUND_ROBIN"`

Defined in: [src/clients/rest/generated/data-contracts.ts:740](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L740)

The strategy to use when the concurrency limit is reached.

***

### maxRuns

> **maxRuns**: `number`

Defined in: [src/clients/rest/generated/data-contracts.ts:738](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L738)

The maximum number of concurrent workflow runs.

#### Format

int32
