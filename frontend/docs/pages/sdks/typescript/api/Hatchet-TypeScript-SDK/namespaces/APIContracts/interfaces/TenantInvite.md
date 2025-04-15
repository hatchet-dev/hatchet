[**Hatchet TypeScript SDK**](../../../../README.md)

***

[Hatchet TypeScript SDK](../../../../README.md) / [APIContracts](../README.md) / TenantInvite

# Interface: TenantInvite

Defined in: [src/clients/rest/generated/data-contracts.ts:501](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L501)

## Properties

### email

> **email**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:504](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L504)

The email of the user to invite.

***

### expires

> **expires**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:515](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L515)

The time that this invite expires.

#### Format

date-time

***

### metadata

> **metadata**: [`APIResourceMeta`](APIResourceMeta.md)

Defined in: [src/clients/rest/generated/data-contracts.ts:502](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L502)

***

### role

> **role**: [`TenantMemberRole`](../enumerations/TenantMemberRole.md)

Defined in: [src/clients/rest/generated/data-contracts.ts:506](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L506)

The role of the user in the tenant.

***

### tenantId

> **tenantId**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:508](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L508)

The tenant id associated with this tenant invite.

***

### tenantName?

> `optional` **tenantName**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:510](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L510)

The tenant name for the tenant.
