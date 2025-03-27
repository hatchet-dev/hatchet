[@hatchet-dev/typescript-sdk](../README.md) / [Exports](../modules.md) / [APIContracts](../modules/APIContracts.md) / APIError

# Interface: APIError

[APIContracts](../modules/APIContracts.md).APIError

## Table of contents

### Properties

- [code](APIContracts.APIError.md#code)
- [description](APIContracts.APIError.md#description)
- [docs\_link](APIContracts.APIError.md#docs_link)
- [field](APIContracts.APIError.md#field)

## Properties

### code

• `Optional` **code**: `number`

a custom Hatchet error code

**`Format`**

uint64

**`Example`**

```ts
1400
```

#### Defined in

[src/clients/rest/generated/data-contracts.ts:34](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L34)

___

### description

• **description**: `string`

a description for this error

**`Example`**

```ts
"A descriptive error message"
```

#### Defined in

[src/clients/rest/generated/data-contracts.ts:44](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L44)

___

### docs\_link

• `Optional` **docs\_link**: `string`

a link to the documentation for this error, if it exists

**`Example`**

```ts
"github.com/hatchet-dev/hatchet"
```

#### Defined in

[src/clients/rest/generated/data-contracts.ts:49](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L49)

___

### field

• `Optional` **field**: `string`

the field that this error is associated with, if applicable

**`Example`**

```ts
"name"
```

#### Defined in

[src/clients/rest/generated/data-contracts.ts:39](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L39)
