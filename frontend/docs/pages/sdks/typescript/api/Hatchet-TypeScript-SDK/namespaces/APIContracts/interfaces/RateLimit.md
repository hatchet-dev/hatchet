[**Hatchet TypeScript SDK**](../../../../README.md)

***

[Hatchet TypeScript SDK](../../../../README.md) / [APIContracts](../README.md) / RateLimit

# Interface: RateLimit

Defined in: [src/clients/rest/generated/data-contracts.ts:681](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L681)

## Properties

### key

> **key**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:683](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L683)

The key for the rate limit.

***

### lastRefill

> **lastRefill**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:697](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L697)

The last time the rate limit was refilled.

#### Format

date-time

#### Example

```ts
"2022-12-13T15:06:48.888358-05:00"
```

***

### limitValue

> **limitValue**: `number`

Defined in: [src/clients/rest/generated/data-contracts.ts:687](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L687)

The maximum number of requests allowed within the window.

***

### tenantId

> **tenantId**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:685](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L685)

The ID of the tenant associated with this rate limit.

***

### value

> **value**: `number`

Defined in: [src/clients/rest/generated/data-contracts.ts:689](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L689)

The current number of requests made within the window.

***

### window

> **window**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:691](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L691)

The window of time in which the limitValue is enforced.
