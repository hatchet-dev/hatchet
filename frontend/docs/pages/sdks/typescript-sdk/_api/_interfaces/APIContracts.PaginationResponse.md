[@hatchet-dev/typescript-sdk](../README.md) / [Exports](../modules.md) / [APIContracts](../modules/APIContracts.md) / PaginationResponse

# Interface: PaginationResponse

[APIContracts](../modules/APIContracts.md).PaginationResponse

**`Example`**

```ts
{"next_page":3,"num_pages":10,"current_page":2}
```

## Table of contents

### Properties

- [current\_page](APIContracts.PaginationResponse.md#current_page)
- [next\_page](APIContracts.PaginationResponse.md#next_page)
- [num\_pages](APIContracts.PaginationResponse.md#num_pages)

## Properties

### current\_page

• `Optional` **current\_page**: `number`

the current page

**`Format`**

int64

**`Example`**

```ts
2
```

#### Defined in

[src/clients/rest/generated/data-contracts.ts:59](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L59)

___

### next\_page

• `Optional` **next\_page**: `number`

the next page

**`Format`**

int64

**`Example`**

```ts
3
```

#### Defined in

[src/clients/rest/generated/data-contracts.ts:65](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L65)

___

### num\_pages

• `Optional` **num\_pages**: `number`

the total number of pages for listing

**`Format`**

int64

**`Example`**

```ts
10
```

#### Defined in

[src/clients/rest/generated/data-contracts.ts:71](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L71)
