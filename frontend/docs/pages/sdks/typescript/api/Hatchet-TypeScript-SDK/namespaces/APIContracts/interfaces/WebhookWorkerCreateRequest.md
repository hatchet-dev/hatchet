[**Hatchet TypeScript SDK**](../../../../README.md)

***

[Hatchet TypeScript SDK](../../../../README.md) / [APIContracts](../README.md) / WebhookWorkerCreateRequest

# Interface: WebhookWorkerCreateRequest

Defined in: [src/clients/rest/generated/data-contracts.ts:1444](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1444)

## Properties

### name

> **name**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1446](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1446)

The name of the webhook worker.

***

### secret?

> `optional` **secret**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1453](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1453)

The secret key for validation. If not provided, a random secret will be generated.

#### Min Length

32

***

### url

> **url**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1448](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1448)

The webhook url.
