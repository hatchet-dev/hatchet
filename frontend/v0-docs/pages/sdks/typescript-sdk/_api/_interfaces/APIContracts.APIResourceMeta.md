[@hatchet-dev/typescript-sdk](../README.md) / [Exports](../modules.md) / [APIContracts](../modules/APIContracts.md) / APIResourceMeta

# Interface: APIResourceMeta

[APIContracts](../modules/APIContracts.md).APIResourceMeta

## Table of contents

### Properties

- [createdAt](APIContracts.APIResourceMeta.md#createdat)
- [id](APIContracts.APIResourceMeta.md#id)
- [updatedAt](APIContracts.APIResourceMeta.md#updatedat)

## Properties

### createdAt

• **createdAt**: `string`

the time that this resource was created

**`Format`**

date-time

**`Example`**

```ts
"2022-12-13T20:06:48.888Z"
```

#### Defined in

[src/clients/rest/generated/data-contracts.ts:88](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L88)

___

### id

• **id**: `string`

the id of this resource, in UUID format

**`Format`**

uuid

**`Min Length`**

36

**`Max Length`**

36

**`Example`**

```ts
"bb214807-246e-43a5-a25d-41761d1cff9e"
```

#### Defined in

[src/clients/rest/generated/data-contracts.ts:82](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L82)

___

### updatedAt

• **updatedAt**: `string`

the time that this resource was last updated

**`Format`**

date-time

**`Example`**

```ts
"2022-12-13T20:06:48.888Z"
```

#### Defined in

[src/clients/rest/generated/data-contracts.ts:94](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L94)
