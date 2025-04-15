[**Hatchet TypeScript SDK**](../../../../README.md)

***

[Hatchet TypeScript SDK](../../../../README.md) / [APIContracts](../README.md) / ScheduledWorkflows

# Interface: ScheduledWorkflows

Defined in: [src/clients/rest/generated/data-contracts.ts:926](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L926)

## Properties

### additionalMetadata?

> `optional` **additionalMetadata**: `Record`\<`string`, `any`\>

Defined in: [src/clients/rest/generated/data-contracts.ts:935](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L935)

***

### input?

> `optional` **input**: `Record`\<`string`, `any`\>

Defined in: [src/clients/rest/generated/data-contracts.ts:934](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L934)

***

### metadata

> **metadata**: [`APIResourceMeta`](APIResourceMeta.md)

Defined in: [src/clients/rest/generated/data-contracts.ts:927](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L927)

***

### method

> **method**: `"DEFAULT"` \| `"API"`

Defined in: [src/clients/rest/generated/data-contracts.ts:947](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L947)

***

### priority?

> `optional` **priority**: `number`

Defined in: [src/clients/rest/generated/data-contracts.ts:953](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L953)

#### Format

int32

#### Min

1

#### Max

3

***

### tenantId

> **tenantId**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:928](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L928)

***

### triggerAt

> **triggerAt**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:933](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L933)

#### Format

date-time

***

### workflowId

> **workflowId**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:930](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L930)

***

### workflowName

> **workflowName**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:931](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L931)

***

### workflowRunCreatedAt?

> `optional` **workflowRunCreatedAt**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:937](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L937)

#### Format

date-time

***

### workflowRunId?

> `optional` **workflowRunId**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:946](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L946)

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

### workflowRunName?

> `optional` **workflowRunName**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:938](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L938)

***

### workflowRunStatus?

> `optional` **workflowRunStatus**: [`WorkflowRunStatus`](../enumerations/WorkflowRunStatus.md)

Defined in: [src/clients/rest/generated/data-contracts.ts:939](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L939)

***

### workflowVersionId

> **workflowVersionId**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:929](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L929)
