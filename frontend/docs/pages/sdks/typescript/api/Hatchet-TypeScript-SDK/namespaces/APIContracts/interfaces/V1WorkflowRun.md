[**Hatchet TypeScript SDK**](../../../../README.md)

***

[Hatchet TypeScript SDK](../../../../README.md) / [APIContracts](../README.md) / V1WorkflowRun

# Interface: V1WorkflowRun

Defined in: [src/clients/rest/generated/data-contracts.ts:1644](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1644)

## Properties

### additionalMetadata?

> `optional` **additionalMetadata**: `object`

Defined in: [src/clients/rest/generated/data-contracts.ts:1668](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1668)

Additional metadata for the task run.

***

### createdAt?

> `optional` **createdAt**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1688](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1688)

The timestamp the task run was created.

#### Format

date-time

***

### displayName

> **displayName**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1670](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1670)

The display name of the task run.

***

### duration?

> `optional` **duration**: `number`

Defined in: [src/clients/rest/generated/data-contracts.ts:1658](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1658)

The duration of the task run, in milliseconds.

***

### errorMessage?

> `optional` **errorMessage**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1676](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1676)

The error message of the task run (for the latest run)

***

### finishedAt?

> `optional` **finishedAt**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1656](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1656)

The timestamp the task run finished.

#### Format

date-time

***

### input

> **input**: `object`

Defined in: [src/clients/rest/generated/data-contracts.ts:1683](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1683)

The input of the task run.

***

### metadata

> **metadata**: [`APIResourceMeta`](APIResourceMeta.md)

Defined in: [src/clients/rest/generated/data-contracts.ts:1645](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1645)

***

### output

> **output**: `object`

Defined in: [src/clients/rest/generated/data-contracts.ts:1674](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1674)

The output of the task run (for the latest run)

***

### parentTaskExternalId?

> `optional` **parentTaskExternalId**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1694](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1694)

#### Format

uuid

#### Min Length

36

#### Max Length

36

***

### startedAt?

> `optional` **startedAt**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1651](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1651)

The timestamp the task run started.

#### Format

date-time

***

### status

> **status**: [`V1TaskStatus`](../enumerations/V1TaskStatus.md)

Defined in: [src/clients/rest/generated/data-contracts.ts:1646](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1646)

***

### tenantId

> **tenantId**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1666](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1666)

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

### workflowId

> **workflowId**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1672](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1672)

#### Format

uuid

***

### workflowVersionId?

> `optional` **workflowVersionId**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1681](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1681)

The ID of the workflow version.

#### Format

uuid
