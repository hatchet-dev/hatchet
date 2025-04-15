[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / RatelimitsClient

# Class: RatelimitsClient

Defined in: [src/v1/client/features/ratelimits.ts:25](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/ratelimits.ts#L25)

RatelimitsClient is used to manage rate limits for the Hatchet

## Constructors

### Constructor

> **new RatelimitsClient**(`client`): `RatelimitsClient`

Defined in: [src/v1/client/features/ratelimits.ts:30](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/ratelimits.ts#L30)

#### Parameters

##### client

[`HatchetClient`](HatchetClient.md)

#### Returns

`RatelimitsClient`

## Properties

### admin

> **admin**: [`AdminClient`](AdminClient.md)

Defined in: [src/v1/client/features/ratelimits.ts:27](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/ratelimits.ts#L27)

***

### api

> **api**: [`Api`](Api.md)\<`unknown`\>

Defined in: [src/v1/client/features/ratelimits.ts:26](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/ratelimits.ts#L26)

***

### tenantId

> **tenantId**: `string`

Defined in: [src/v1/client/features/ratelimits.ts:28](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/ratelimits.ts#L28)

## Methods

### list()

> **list**(`opts`): `Promise`\<[`RateLimitList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/RateLimitList.md)\>

Defined in: [src/v1/client/features/ratelimits.ts:41](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/ratelimits.ts#L41)

#### Parameters

##### opts

`undefined` |

\{ `limit`: `number`; `offset`: `number`; `orderByDirection`: [`RateLimitOrderByDirection`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/enumerations/RateLimitOrderByDirection.md); `orderByField`: [`RateLimitOrderByField`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/enumerations/RateLimitOrderByField.md); `search`: `string`; \}

###### limit?

`number`

The number to limit by

**Format**

int64

###### offset?

`number`

The number to skip

**Format**

int64

###### orderByDirection?

[`RateLimitOrderByDirection`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/enumerations/RateLimitOrderByDirection.md)

The order direction

###### orderByField?

[`RateLimitOrderByField`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/enumerations/RateLimitOrderByField.md)

What to order by

###### search?

`string`

The search query to filter for

#### Returns

`Promise`\<[`RateLimitList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/RateLimitList.md)\>

***

### upsert()

> **upsert**(`opts`): `Promise`\<`string`\>

Defined in: [src/v1/client/features/ratelimits.ts:36](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/ratelimits.ts#L36)

#### Parameters

##### opts

[`CreateRateLimitOpts`](../type-aliases/CreateRateLimitOpts.md)

#### Returns

`Promise`\<`string`\>
