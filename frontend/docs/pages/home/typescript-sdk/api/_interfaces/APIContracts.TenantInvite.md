[@hatchet-dev/typescript-sdk](../README.md) / [Exports](../modules.md) / [APIContracts](../modules/APIContracts.md) / TenantInvite

# Interface: TenantInvite

[APIContracts](../modules/APIContracts.md).TenantInvite

## Table of contents

### Properties

- [email](APIContracts.TenantInvite.md#email)
- [expires](APIContracts.TenantInvite.md#expires)
- [metadata](APIContracts.TenantInvite.md#metadata)
- [role](APIContracts.TenantInvite.md#role)
- [tenantId](APIContracts.TenantInvite.md#tenantid)
- [tenantName](APIContracts.TenantInvite.md#tenantname)

## Properties

### email

• **email**: `string`

The email of the user to invite.

#### Defined in

[src/clients/rest/generated/data-contracts.ts:191](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L191)

___

### expires

• **expires**: `string`

The time that this invite expires.

**`Format`**

date-time

#### Defined in

[src/clients/rest/generated/data-contracts.ts:202](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L202)

___

### metadata

• **metadata**: [`APIResourceMeta`](APIContracts.APIResourceMeta.md)

#### Defined in

[src/clients/rest/generated/data-contracts.ts:189](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L189)

___

### role

• **role**: [`TenantMemberRole`](../enums/APIContracts.TenantMemberRole.md)

The role of the user in the tenant.

#### Defined in

[src/clients/rest/generated/data-contracts.ts:193](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L193)

___

### tenantId

• **tenantId**: `string`

The tenant id associated with this tenant invite.

#### Defined in

[src/clients/rest/generated/data-contracts.ts:195](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L195)

___

### tenantName

• `Optional` **tenantName**: `string`

The tenant name for the tenant.

#### Defined in

[src/clients/rest/generated/data-contracts.ts:197](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L197)
