[**Hatchet TypeScript SDK**](../../../../README.md)

***

[Hatchet TypeScript SDK](../../../../README.md) / [APIContracts](../README.md) / WorkflowRun

# Interface: WorkflowRun

Defined in: [src/clients/rest/generated/data-contracts.ts:843](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L843)

## Properties

### additionalMetadata?

> `optional` **additionalMetadata**: `Record`\<`string`, `any`\>

Defined in: [src/clients/rest/generated/data-contracts.ts:874](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L874)

***

### displayName?

> `optional` **displayName**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:849](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L849)

***

### duration?

> `optional` **duration**: `number`

Defined in: [src/clients/rest/generated/data-contracts.ts:859](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L859)

#### Example

```ts
1000
```

***

### error?

> `optional` **error**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:853](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L853)

***

### finishedAt?

> `optional` **finishedAt**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:857](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L857)

#### Format

date-time

***

### input?

> `optional` **input**: `Record`\<`string`, `any`\>

Defined in: [src/clients/rest/generated/data-contracts.ts:852](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L852)

***

### jobRuns?

> `optional` **jobRuns**: [`JobRun`](JobRun.md)[]

Defined in: [src/clients/rest/generated/data-contracts.ts:850](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L850)

***

### metadata

> **metadata**: [`APIResourceMeta`](APIResourceMeta.md)

Defined in: [src/clients/rest/generated/data-contracts.ts:844](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L844)

***

### parentId?

> `optional` **parentId**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:866](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L866)

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

Defined in: [src/clients/rest/generated/data-contracts.ts:873](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L873)

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

Defined in: [src/clients/rest/generated/data-contracts.ts:855](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L855)

#### Format

date-time

***

### status

> **status**: [`WorkflowRunStatus`](../enumerations/WorkflowRunStatus.md)

Defined in: [src/clients/rest/generated/data-contracts.ts:848](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L848)

***

### tenantId

> **tenantId**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:845](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L845)

***

### triggeredBy

> **triggeredBy**: [`WorkflowRunTriggeredBy`](WorkflowRunTriggeredBy.md)

Defined in: [src/clients/rest/generated/data-contracts.ts:851](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L851)

***

### workflowVersion?

> `optional` **workflowVersion**: [`WorkflowVersion`](WorkflowVersion.md)

Defined in: [src/clients/rest/generated/data-contracts.ts:847](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L847)

***

### workflowVersionId

> **workflowVersionId**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:846](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L846)
