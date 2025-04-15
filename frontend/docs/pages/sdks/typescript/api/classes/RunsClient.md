[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / RunsClient

# Class: RunsClient

Defined in: [src/v1/client/features/runs.ts:58](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/runs.ts#L58)

RunsClient is used to list and manage runs

## Constructors

### Constructor

> **new RunsClient**(`client`): `RunsClient`

Defined in: [src/v1/client/features/runs.ts:63](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/runs.ts#L63)

#### Parameters

##### client

[`HatchetClient`](HatchetClient.md)

#### Returns

`RunsClient`

## Properties

### api

> **api**: [`Api`](Api.md)\<`unknown`\>

Defined in: [src/v1/client/features/runs.ts:59](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/runs.ts#L59)

***

### tenantId

> **tenantId**: `string`

Defined in: [src/v1/client/features/runs.ts:60](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/runs.ts#L60)

***

### workflows

> **workflows**: [`WorkflowsClient`](WorkflowsClient.md)

Defined in: [src/v1/client/features/runs.ts:61](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/runs.ts#L61)

## Methods

### cancel()

> **cancel**(`opts`): `Promise`\<`AxiosResponse`\<`void`, `any`\>\>

Defined in: [src/v1/client/features/runs.ts:83](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/runs.ts#L83)

#### Parameters

##### opts

[`CancelRunOpts`](../type-aliases/CancelRunOpts.md)

#### Returns

`Promise`\<`AxiosResponse`\<`void`, `any`\>\>

***

### get()

> **get**\<`T`\>(`run`): `Promise`\<[`V1WorkflowRunDetails`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/V1WorkflowRunDetails.md)\>

Defined in: [src/v1/client/features/runs.ts:69](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/runs.ts#L69)

#### Type Parameters

##### T

`T` = `any`

#### Parameters

##### run

`string` | `WorkflowRunRef`\<`T`\>

#### Returns

`Promise`\<[`V1WorkflowRunDetails`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/V1WorkflowRunDetails.md)\>

***

### list()

> **list**(`opts?`): `Promise`\<[`V1TaskSummaryList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/V1TaskSummaryList.md)\>

Defined in: [src/v1/client/features/runs.ts:76](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/runs.ts#L76)

#### Parameters

##### opts?

`Partial`\<[`ListRunsOpts`](../interfaces/ListRunsOpts.md)\>

#### Returns

`Promise`\<[`V1TaskSummaryList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/V1TaskSummaryList.md)\>

***

### replay()

> **replay**(`opts`): `Promise`\<`AxiosResponse`\<`void`, `any`\>\>

Defined in: [src/v1/client/features/runs.ts:92](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/runs.ts#L92)

#### Parameters

##### opts

[`ReplayRunOpts`](../type-aliases/ReplayRunOpts.md)

#### Returns

`Promise`\<`AxiosResponse`\<`void`, `any`\>\>
