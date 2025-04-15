[**Hatchet TypeScript SDK**](../../../../README.md)

***

[Hatchet TypeScript SDK](../../../../README.md) / [APIContracts](../README.md) / UpdateTenantRequest

# Interface: UpdateTenantRequest

Defined in: [src/clients/rest/generated/data-contracts.ts:573](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L573)

## Properties

### alertMemberEmails?

> `optional` **alertMemberEmails**: `boolean`

Defined in: [src/clients/rest/generated/data-contracts.ts:579](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L579)

Whether to alert tenant members.

***

### analyticsOptOut?

> `optional` **analyticsOptOut**: `boolean`

Defined in: [src/clients/rest/generated/data-contracts.ts:577](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L577)

Whether the tenant has opted out of analytics.

***

### enableExpiringTokenAlerts?

> `optional` **enableExpiringTokenAlerts**: `boolean`

Defined in: [src/clients/rest/generated/data-contracts.ts:583](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L583)

Whether to enable alerts when tokens are approaching expiration.

***

### enableTenantResourceLimitAlerts?

> `optional` **enableTenantResourceLimitAlerts**: `boolean`

Defined in: [src/clients/rest/generated/data-contracts.ts:585](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L585)

Whether to enable alerts when tenant resources are approaching limits.

***

### enableWorkflowRunFailureAlerts?

> `optional` **enableWorkflowRunFailureAlerts**: `boolean`

Defined in: [src/clients/rest/generated/data-contracts.ts:581](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L581)

Whether to send alerts when workflow runs fail.

***

### maxAlertingFrequency?

> `optional` **maxAlertingFrequency**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:587](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L587)

The max frequency at which to alert.

***

### name?

> `optional` **name**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:575](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L575)

The name of the tenant.

***

### version?

> `optional` **version**: `any`

Defined in: [src/clients/rest/generated/data-contracts.ts:589](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L589)

The version of the tenant.
