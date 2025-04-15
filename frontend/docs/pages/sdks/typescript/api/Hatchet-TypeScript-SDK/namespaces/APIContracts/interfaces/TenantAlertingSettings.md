[**Hatchet TypeScript SDK**](../../../../README.md)

***

[Hatchet TypeScript SDK](../../../../README.md) / [APIContracts](../README.md) / TenantAlertingSettings

# Interface: TenantAlertingSettings

Defined in: [src/clients/rest/generated/data-contracts.ts:461](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L461)

## Properties

### alertMemberEmails?

> `optional` **alertMemberEmails**: `boolean`

Defined in: [src/clients/rest/generated/data-contracts.ts:464](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L464)

Whether to alert tenant members.

***

### enableExpiringTokenAlerts?

> `optional` **enableExpiringTokenAlerts**: `boolean`

Defined in: [src/clients/rest/generated/data-contracts.ts:468](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L468)

Whether to enable alerts when tokens are approaching expiration.

***

### enableTenantResourceLimitAlerts?

> `optional` **enableTenantResourceLimitAlerts**: `boolean`

Defined in: [src/clients/rest/generated/data-contracts.ts:470](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L470)

Whether to enable alerts when tenant resources are approaching limits.

***

### enableWorkflowRunFailureAlerts?

> `optional` **enableWorkflowRunFailureAlerts**: `boolean`

Defined in: [src/clients/rest/generated/data-contracts.ts:466](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L466)

Whether to send alerts when workflow runs fail.

***

### lastAlertedAt?

> `optional` **lastAlertedAt**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:477](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L477)

The last time an alert was sent.

#### Format

date-time

***

### maxAlertingFrequency

> **maxAlertingFrequency**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:472](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L472)

The max frequency at which to alert.

***

### metadata

> **metadata**: [`APIResourceMeta`](APIResourceMeta.md)

Defined in: [src/clients/rest/generated/data-contracts.ts:462](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L462)
