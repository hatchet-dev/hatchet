[**Hatchet TypeScript SDK**](../../../../README.md)

***

[Hatchet TypeScript SDK](../../../../README.md) / [APIContracts](../README.md) / V1WorkflowRunDetails

# Interface: V1WorkflowRunDetails

Defined in: [src/clients/rest/generated/data-contracts.ts:1697](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1697)

## Properties

### run

> **run**: [`V1WorkflowRun`](V1WorkflowRun.md)

Defined in: [src/clients/rest/generated/data-contracts.ts:1698](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1698)

***

### shape

> **shape**: `object`[]

Defined in: [src/clients/rest/generated/data-contracts.ts:1701](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1701)

#### childrenStepIds

> **childrenStepIds**: `string`[]

#### stepId

> **stepId**: `string`

##### Format

uuid

##### Min Length

36

##### Max Length

36

#### taskExternalId

> **taskExternalId**: `string`

##### Format

uuid

##### Min Length

36

##### Max Length

36

#### taskName

> **taskName**: `string`

***

### taskEvents

> **taskEvents**: `any`[]

Defined in: [src/clients/rest/generated/data-contracts.ts:1700](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1700)

The list of task events for the workflow run

***

### tasks

> **tasks**: [`V1TaskSummary`](V1TaskSummary.md)[]

Defined in: [src/clients/rest/generated/data-contracts.ts:1717](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1717)
