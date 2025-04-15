[**Hatchet TypeScript SDK**](../../../../README.md)

***

[Hatchet TypeScript SDK](../../../../README.md) / [APIContracts](../README.md) / WorkflowVersion

# Interface: WorkflowVersion

Defined in: [src/clients/rest/generated/data-contracts.ts:755](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L755)

## Properties

### concurrency?

> `optional` **concurrency**: [`WorkflowConcurrency`](WorkflowConcurrency.md)

Defined in: [src/clients/rest/generated/data-contracts.ts:770](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L770)

***

### defaultPriority?

> `optional` **defaultPriority**: `number`

Defined in: [src/clients/rest/generated/data-contracts.ts:768](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L768)

The default priority of the workflow.

#### Format

int32

***

### jobs?

> `optional` **jobs**: [`Job`](Job.md)[]

Defined in: [src/clients/rest/generated/data-contracts.ts:773](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L773)

***

### metadata

> **metadata**: [`APIResourceMeta`](APIResourceMeta.md)

Defined in: [src/clients/rest/generated/data-contracts.ts:756](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L756)

***

### order

> **order**: `number`

Defined in: [src/clients/rest/generated/data-contracts.ts:760](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L760)

#### Format

int32

***

### scheduleTimeout?

> `optional` **scheduleTimeout**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:772](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L772)

***

### sticky?

> `optional` **sticky**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:763](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L763)

The sticky strategy of the workflow.

***

### triggers?

> `optional` **triggers**: [`WorkflowTriggers`](WorkflowTriggers.md)

Defined in: [src/clients/rest/generated/data-contracts.ts:771](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L771)

***

### version

> **version**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:758](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L758)

The version of the workflow.

***

### workflow?

> `optional` **workflow**: [`Workflow`](Workflow.md)

Defined in: [src/clients/rest/generated/data-contracts.ts:769](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L769)

***

### workflowId

> **workflowId**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:761](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L761)
