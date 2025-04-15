[**Hatchet TypeScript SDK**](../../../../README.md)

***

[Hatchet TypeScript SDK](../../../../README.md) / [APIContracts](../README.md) / WorkflowRunShape

# Interface: WorkflowRunShape

Defined in: [src/clients/rest/generated/data-contracts.ts:877](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L877)

## Properties

### additionalMetadata?

> `optional` **additionalMetadata**: `Record`\<`string`, `any`\>

Defined in: [src/clients/rest/generated/data-contracts.ts:909](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L909)

***

### displayName?

> `optional` **displayName**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:884](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L884)

***

### duration?

> `optional` **duration**: `number`

Defined in: [src/clients/rest/generated/data-contracts.ts:894](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L894)

#### Example

```ts
1000
```

***

### error?

> `optional` **error**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:888](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L888)

***

### finishedAt?

> `optional` **finishedAt**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:892](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L892)

#### Format

date-time

***

### input?

> `optional` **input**: `Record`\<`string`, `any`\>

Defined in: [src/clients/rest/generated/data-contracts.ts:887](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L887)

***

### jobRuns?

> `optional` **jobRuns**: [`JobRun`](JobRun.md)[]

Defined in: [src/clients/rest/generated/data-contracts.ts:885](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L885)

***

### metadata

> **metadata**: [`APIResourceMeta`](APIResourceMeta.md)

Defined in: [src/clients/rest/generated/data-contracts.ts:878](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L878)

***

### parentId?

> `optional` **parentId**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:901](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L901)

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

### parentStepRunId?

> `optional` **parentStepRunId**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:908](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L908)

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

### startedAt?

> `optional` **startedAt**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:890](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L890)

#### Format

date-time

***

### status

> **status**: [`WorkflowRunStatus`](../enumerations/WorkflowRunStatus.md)

Defined in: [src/clients/rest/generated/data-contracts.ts:883](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L883)

***

### tenantId

> **tenantId**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:879](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L879)

***

### triggeredBy

> **triggeredBy**: [`WorkflowRunTriggeredBy`](WorkflowRunTriggeredBy.md)

Defined in: [src/clients/rest/generated/data-contracts.ts:886](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L886)

***

### workflowId?

> `optional` **workflowId**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:880](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L880)

***

### workflowVersion?

> `optional` **workflowVersion**: [`WorkflowVersion`](WorkflowVersion.md)

Defined in: [src/clients/rest/generated/data-contracts.ts:882](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L882)

***

### workflowVersionId

> **workflowVersionId**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:881](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L881)
