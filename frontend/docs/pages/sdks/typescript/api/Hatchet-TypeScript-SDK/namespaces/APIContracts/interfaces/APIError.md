[**Hatchet TypeScript SDK**](../../../../README.md)

***

[Hatchet TypeScript SDK](../../../../README.md) / [APIContracts](../README.md) / APIError

# Interface: APIError

Defined in: [src/clients/rest/generated/data-contracts.ts:267](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L267)

## Properties

### code?

> `optional` **code**: `number`

Defined in: [src/clients/rest/generated/data-contracts.ts:273](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L273)

a custom Hatchet error code

#### Format

uint64

#### Example

```ts
1400
```

***

### description

> **description**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:283](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L283)

a description for this error

#### Example

```ts
"A descriptive error message"
```

***

### docs\_link?

> `optional` **docs\_link**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:288](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L288)

a link to the documentation for this error, if it exists

#### Example

```ts
"github.com/hatchet-dev/hatchet"
```

***

### field?

> `optional` **field**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:278](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L278)

the field that this error is associated with, if applicable

#### Example

```ts
"name"
```
