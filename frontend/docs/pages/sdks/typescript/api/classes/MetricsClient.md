[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / MetricsClient

# Class: MetricsClient

Defined in: [src/v1/client/features/metrics.ts:7](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/metrics.ts#L7)

MetricsClient is used to get metrics for workflows

## Constructors

### Constructor

> **new MetricsClient**(`client`): `MetricsClient`

Defined in: [src/v1/client/features/metrics.ts:11](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/metrics.ts#L11)

#### Parameters

##### client

[`HatchetClient`](HatchetClient.md)

#### Returns

`MetricsClient`

## Properties

### api

> **api**: [`Api`](Api.md)\<`unknown`\>

Defined in: [src/v1/client/features/metrics.ts:9](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/metrics.ts#L9)

***

### tenantId

> **tenantId**: `string`

Defined in: [src/v1/client/features/metrics.ts:8](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/metrics.ts#L8)

## Methods

### getQueueMetrics()

> **getQueueMetrics**(`opts?`): `Promise`\<[`TenantQueueMetrics`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/TenantQueueMetrics.md)\>

Defined in: [src/v1/client/features/metrics.ts:26](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/metrics.ts#L26)

#### Parameters

##### opts?

###### additionalMetadata?

`string`[]

A list of metadata key value pairs to filter by

**Example**

```ts
["key1:value1","key2:value2"]
```

###### workflows?

`string`[]

A list of workflow IDs to filter by

#### Returns

`Promise`\<[`TenantQueueMetrics`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/TenantQueueMetrics.md)\>

***

### getTaskMetrics()

> **getTaskMetrics**(`opts?`): `Promise`\<[`TenantStepRunQueueMetrics`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/TenantStepRunQueueMetrics.md)\>

Defined in: [src/v1/client/features/metrics.ts:42](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/metrics.ts#L42)

#### Parameters

##### opts?

`RequestParams`

#### Returns

`Promise`\<[`TenantStepRunQueueMetrics`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/TenantStepRunQueueMetrics.md)\>

***

### getWorkflowMetrics()

> **getWorkflowMetrics**(`workflow`, `opts?`): `Promise`\<[`WorkflowMetrics`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowMetrics.md)\>

Defined in: [src/v1/client/features/metrics.ts:16](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/metrics.ts#L16)

#### Parameters

##### workflow

`string` | [`Workflow`](../interfaces/Workflow.md) | [`BaseWorkflowDeclaration`](BaseWorkflowDeclaration.md)\<`any`, `any`\>

##### opts?

###### groupKey?

`string`

A group key to filter metrics by

###### status?

[`WorkflowRunStatus`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/enumerations/WorkflowRunStatus.md)

A status of workflow run statuses to filter by

#### Returns

`Promise`\<[`WorkflowMetrics`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowMetrics.md)\>
