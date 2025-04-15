[**Hatchet TypeScript SDK**](../../../../README.md)

***

[Hatchet TypeScript SDK](../../../../README.md) / [APIContracts](../README.md) / V1TaskSummary

# Interface: V1TaskSummary

Defined in: [src/clients/rest/generated/data-contracts.ts:1480](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1480)

## Properties

### actionId?

> `optional` **actionId**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1483](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1483)

The action ID of the task.

***

### additionalMetadata?

> `optional` **additionalMetadata**: `object`

Defined in: [src/clients/rest/generated/data-contracts.ts:1485](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1485)

Additional metadata for the task run.

***

### children?

> `optional` **children**: `V1TaskSummary`[]

Defined in: [src/clients/rest/generated/data-contracts.ts:1487](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1487)

The list of children tasks

***

### createdAt

> **createdAt**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1492](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1492)

The timestamp the task was created.

#### Format

date-time

***

### displayName

> **displayName**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1494](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1494)

The display name of the task run.

***

### duration?

> `optional` **duration**: `number`

Defined in: [src/clients/rest/generated/data-contracts.ts:1496](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1496)

The duration of the task run, in milliseconds.

***

### errorMessage?

> `optional` **errorMessage**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1498](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1498)

The error message of the task run (for the latest run)

***

### finishedAt?

> `optional` **finishedAt**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1503](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1503)

The timestamp the task run finished.

#### Format

date-time

***

### input

> **input**: `object`

Defined in: [src/clients/rest/generated/data-contracts.ts:1505](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1505)

The input of the task run.

***

### metadata

> **metadata**: [`APIResourceMeta`](APIResourceMeta.md)

Defined in: [src/clients/rest/generated/data-contracts.ts:1481](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1481)

***

### numSpawnedChildren

> **numSpawnedChildren**: `number`

Defined in: [src/clients/rest/generated/data-contracts.ts:1507](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1507)

The number of spawned children tasks

***

### output

> **output**: `object`

Defined in: [src/clients/rest/generated/data-contracts.ts:1509](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1509)

The output of the task run (for the latest run)

***

### startedAt?

> `optional` **startedAt**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1515](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1515)

The timestamp the task run started.

#### Format

date-time

***

### status

> **status**: [`V1TaskStatus`](../enumerations/V1TaskStatus.md)

Defined in: [src/clients/rest/generated/data-contracts.ts:1510](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1510)

***

### stepId?

> `optional` **stepId**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1522](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1522)

The step ID of the task.

#### Format

uuid

#### Min Length

36

#### Max Length

36

***

### taskExternalId

> **taskExternalId**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1529](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1529)

The external ID of the task.

#### Format

uuid

#### Min Length

36

#### Max Length

36

***

### taskId

> **taskId**: `number`

Defined in: [src/clients/rest/generated/data-contracts.ts:1531](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1531)

The ID of the task.

***

### taskInsertedAt

> **taskInsertedAt**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1536](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1536)

The timestamp the task was inserted.

#### Format

date-time

***

### tenantId

> **tenantId**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1544](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1544)

The ID of the tenant.

#### Format

uuid

#### Min Length

36

#### Max Length

36

#### Example

```ts
"bb214807-246e-43a5-a25d-41761d1cff9e"
```

***

### type

> **type**: `"DAG"` \| `"TASK"`

Defined in: [src/clients/rest/generated/data-contracts.ts:1546](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1546)

The type of the workflow (whether it's a DAG or a task)

***

### workflowId

> **workflowId**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1548](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1548)

#### Format

uuid

***

### workflowName?

> `optional` **workflowName**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1549](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1549)

***

### workflowRunExternalId?

> `optional` **workflowRunExternalId**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1554](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1554)

The external ID of the workflow run

#### Format

uuid

***

### workflowVersionId?

> `optional` **workflowVersionId**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1559](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1559)

The version ID of the workflow

#### Format

uuid
