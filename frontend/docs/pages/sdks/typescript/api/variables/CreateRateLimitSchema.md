[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / CreateRateLimitSchema

# Variable: CreateRateLimitSchema

> `const` **CreateRateLimitSchema**: `ZodObject`\<\{ `duration`: `ZodOptional`\<`ZodNativeEnum`\<*typeof* [`RateLimitDuration`](../enumerations/RateLimitDuration.md)\>\>; `dynamicKey`: `ZodOptional`\<`ZodString`\>; `key`: `ZodOptional`\<`ZodString`\>; `limit`: `ZodOptional`\<`ZodUnion`\<\[`ZodNumber`, `ZodString`\]\>\>; `staticKey`: `ZodOptional`\<`ZodString`\>; `units`: `ZodUnion`\<\[`ZodNumber`, `ZodString`\]\>; \}, `"strip"`, `ZodTypeAny`, \{ `duration`: [`SECOND`](../enumerations/RateLimitDuration.md#second) \| [`MINUTE`](../enumerations/RateLimitDuration.md#minute) \| [`HOUR`](../enumerations/RateLimitDuration.md#hour) \| [`DAY`](../enumerations/RateLimitDuration.md#day) \| [`WEEK`](../enumerations/RateLimitDuration.md#week) \| [`MONTH`](../enumerations/RateLimitDuration.md#month) \| [`YEAR`](../enumerations/RateLimitDuration.md#year) \| [`UNRECOGNIZED`](../enumerations/RateLimitDuration.md#unrecognized); `dynamicKey`: `string`; `key`: `string`; `limit`: `string` \| `number`; `staticKey`: `string`; `units`: `string` \| `number`; \}, \{ `duration`: [`SECOND`](../enumerations/RateLimitDuration.md#second) \| [`MINUTE`](../enumerations/RateLimitDuration.md#minute) \| [`HOUR`](../enumerations/RateLimitDuration.md#hour) \| [`DAY`](../enumerations/RateLimitDuration.md#day) \| [`WEEK`](../enumerations/RateLimitDuration.md#week) \| [`MONTH`](../enumerations/RateLimitDuration.md#month) \| [`YEAR`](../enumerations/RateLimitDuration.md#year) \| [`UNRECOGNIZED`](../enumerations/RateLimitDuration.md#unrecognized); `dynamicKey`: `string`; `key`: `string`; `limit`: `string` \| `number`; `staticKey`: `string`; `units`: `string` \| `number`; \}\>

Defined in: [src/step.ts:27](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L27)
