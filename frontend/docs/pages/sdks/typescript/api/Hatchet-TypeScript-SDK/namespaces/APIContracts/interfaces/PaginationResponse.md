[**Hatchet TypeScript SDK**](../../../../README.md)

***

[Hatchet TypeScript SDK](../../../../README.md) / [APIContracts](../README.md) / PaginationResponse

# Interface: PaginationResponse

Defined in: [src/clients/rest/generated/data-contracts.ts:292](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L292)

## Example

```ts
{"next_page":3,"num_pages":10,"current_page":2}
```

## Properties

### current\_page?

> `optional` **current\_page**: `number`

Defined in: [src/clients/rest/generated/data-contracts.ts:298](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L298)

the current page

#### Format

int64

#### Example

```ts
2
```

***

### next\_page?

> `optional` **next\_page**: `number`

Defined in: [src/clients/rest/generated/data-contracts.ts:304](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L304)

the next page

#### Format

int64

#### Example

```ts
3
```

***

### num\_pages?

> `optional` **num\_pages**: `number`

Defined in: [src/clients/rest/generated/data-contracts.ts:310](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L310)

the total number of pages for listing

#### Format

int64

#### Example

```ts
10
```
