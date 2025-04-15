[**Hatchet TypeScript SDK**](../../../../README.md)

***

[Hatchet TypeScript SDK](../../../../README.md) / [APIContracts](../README.md) / APIMeta

# Interface: APIMeta

Defined in: [src/clients/rest/generated/data-contracts.ts:200](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L200)

## Properties

### allowChangePassword?

> `optional` **allowChangePassword**: `boolean`

Defined in: [src/clients/rest/generated/data-contracts.ts:227](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L227)

whether or not users can change their password

#### Example

```ts
true
```

***

### allowCreateTenant?

> `optional` **allowCreateTenant**: `boolean`

Defined in: [src/clients/rest/generated/data-contracts.ts:222](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L222)

whether or not users can create new tenants

#### Example

```ts
true
```

***

### allowInvites?

> `optional` **allowInvites**: `boolean`

Defined in: [src/clients/rest/generated/data-contracts.ts:217](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L217)

whether or not users can invite other users to this instance

#### Example

```ts
true
```

***

### allowSignup?

> `optional` **allowSignup**: `boolean`

Defined in: [src/clients/rest/generated/data-contracts.ts:212](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L212)

whether or not users can sign up for this instance

#### Example

```ts
true
```

***

### auth?

> `optional` **auth**: [`APIMetaAuth`](APIMetaAuth.md)

Defined in: [src/clients/rest/generated/data-contracts.ts:201](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L201)

***

### posthog?

> `optional` **posthog**: [`APIMetaPosthog`](APIMetaPosthog.md)

Defined in: [src/clients/rest/generated/data-contracts.ts:207](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L207)

***

### pylonAppId?

> `optional` **pylonAppId**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:206](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L206)

the Pylon app ID for usepylon.com chat support

#### Example

```ts
"12345678-1234-1234-1234-123456789012"
```
