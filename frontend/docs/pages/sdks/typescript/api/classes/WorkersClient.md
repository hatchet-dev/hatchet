[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / WorkersClient

# Class: WorkersClient

Defined in: [src/v1/client/features/workers.ts:6](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/workers.ts#L6)

WorkersClient is used to list and manage workers

## Constructors

### Constructor

> **new WorkersClient**(`client`): `WorkersClient`

Defined in: [src/v1/client/features/workers.ts:10](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/workers.ts#L10)

#### Parameters

##### client

[`HatchetClient`](HatchetClient.md)

#### Returns

`WorkersClient`

## Properties

### api

> **api**: [`Api`](Api.md)\<`unknown`\>

Defined in: [src/v1/client/features/workers.ts:7](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/workers.ts#L7)

***

### tenantId

> **tenantId**: `string`

Defined in: [src/v1/client/features/workers.ts:8](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/workers.ts#L8)

## Methods

### get()

> **get**(`workerId`): `Promise`\<[`Worker`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/Worker.md)\>

Defined in: [src/v1/client/features/workers.ts:15](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/workers.ts#L15)

#### Parameters

##### workerId

`string`

#### Returns

`Promise`\<[`Worker`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/Worker.md)\>

***

### isPaused()

> **isPaused**(`workerId`): `Promise`\<`boolean`\>

Defined in: [src/v1/client/features/workers.ts:25](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/workers.ts#L25)

#### Parameters

##### workerId

`string`

#### Returns

`Promise`\<`boolean`\>

***

### list()

> **list**(): `Promise`\<[`WorkerList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkerList.md)\>

Defined in: [src/v1/client/features/workers.ts:20](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/workers.ts#L20)

#### Returns

`Promise`\<[`WorkerList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkerList.md)\>

***

### pause()

> **pause**(`workerId`): `Promise`\<[`Worker`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/Worker.md)\>

Defined in: [src/v1/client/features/workers.ts:30](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/workers.ts#L30)

#### Parameters

##### workerId

`string`

#### Returns

`Promise`\<[`Worker`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/Worker.md)\>

***

### unpause()

> **unpause**(`workerId`): `Promise`\<[`Worker`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/Worker.md)\>

Defined in: [src/v1/client/features/workers.ts:37](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/workers.ts#L37)

#### Parameters

##### workerId

`string`

#### Returns

`Promise`\<[`Worker`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/Worker.md)\>
