[**Hatchet TypeScript SDK**](../../../../README.md)

***

[Hatchet TypeScript SDK](../../../../README.md) / [APIContracts](../README.md) / TenantMember

# Interface: TenantMember

Defined in: [src/clients/rest/generated/data-contracts.ts:410](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L410)

## Properties

### metadata

> **metadata**: [`APIResourceMeta`](APIResourceMeta.md)

Defined in: [src/clients/rest/generated/data-contracts.ts:411](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L411)

***

### role

> **role**: [`TenantMemberRole`](../enumerations/TenantMemberRole.md)

Defined in: [src/clients/rest/generated/data-contracts.ts:415](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L415)

The role of the user in the tenant.

***

### tenant?

> `optional` **tenant**: [`Tenant`](Tenant.md)

Defined in: [src/clients/rest/generated/data-contracts.ts:417](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L417)

The tenant associated with this tenant member.

***

### user

> **user**: [`UserTenantPublic`](UserTenantPublic.md)

Defined in: [src/clients/rest/generated/data-contracts.ts:413](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L413)

The user associated with this tenant member.
