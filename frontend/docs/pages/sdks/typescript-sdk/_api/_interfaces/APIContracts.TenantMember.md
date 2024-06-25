[@hatchet-dev/typescript-sdk](../README.md) / [Exports](../modules.md) / [APIContracts](../modules/APIContracts.md) / TenantMember

# Interface: TenantMember

[APIContracts](../modules/APIContracts.md).TenantMember

## Table of contents

### Properties

- [metadata](APIContracts.TenantMember.md#metadata)
- [role](APIContracts.TenantMember.md#role)
- [tenant](APIContracts.TenantMember.md#tenant)
- [user](APIContracts.TenantMember.md#user)

## Properties

### metadata

• **metadata**: [`APIResourceMeta`](APIContracts.APIResourceMeta.md)

#### Defined in

[src/clients/rest/generated/data-contracts.ts:156](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L156)

___

### role

• **role**: [`TenantMemberRole`](../enums/APIContracts.TenantMemberRole.md)

The role of the user in the tenant.

#### Defined in

[src/clients/rest/generated/data-contracts.ts:160](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L160)

___

### tenant

• `Optional` **tenant**: [`Tenant`](APIContracts.Tenant.md)

The tenant associated with this tenant member.

#### Defined in

[src/clients/rest/generated/data-contracts.ts:162](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L162)

___

### user

• **user**: [`UserTenantPublic`](APIContracts.UserTenantPublic.md)

The user associated with this tenant member.

#### Defined in

[src/clients/rest/generated/data-contracts.ts:158](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L158)
