[**Hatchet TypeScript SDK**](../../../../README.md)

***

[Hatchet TypeScript SDK](../../../../README.md) / [APIContracts](../README.md) / TenantResourceLimit

# Interface: TenantResourceLimit

Defined in: [src/clients/rest/generated/data-contracts.ts:425](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L425)

## Properties

### alarmValue?

> `optional` **alarmValue**: `number`

Defined in: [src/clients/rest/generated/data-contracts.ts:432](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L432)

The alarm value associated with this limit to warn of approaching limit value.

***

### lastRefill?

> `optional` **lastRefill**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:441](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L441)

The last time the limit was refilled.

#### Format

date-time

***

### limitValue

> **limitValue**: `number`

Defined in: [src/clients/rest/generated/data-contracts.ts:430](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L430)

The limit associated with this limit.

***

### metadata

> **metadata**: [`APIResourceMeta`](APIResourceMeta.md)

Defined in: [src/clients/rest/generated/data-contracts.ts:426](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L426)

***

### resource

> **resource**: [`TenantResource`](../enumerations/TenantResource.md)

Defined in: [src/clients/rest/generated/data-contracts.ts:428](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L428)

The resource associated with this limit.

***

### value

> **value**: `number`

Defined in: [src/clients/rest/generated/data-contracts.ts:434](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L434)

The current value associated with this limit.

***

### window?

> `optional` **window**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:436](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L436)

The meter window for the limit. (i.e. 1 day, 1 week, 1 month)
