[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / DesiredWorkerLabelSchema

# Variable: DesiredWorkerLabelSchema

> `const` **DesiredWorkerLabelSchema**: `ZodOptional`\<`ZodUnion`\<\[`ZodString`, `ZodNumber`, `ZodObject`\<\{ `comparator`: `ZodOptional`\<`ZodNativeEnum`\<*typeof* `WorkerLabelComparator`\>\>; `required`: `ZodOptional`\<`ZodBoolean`\>; `value`: `ZodUnion`\<\[`ZodString`, `ZodNumber`\]\>; `weight`: `ZodOptional`\<`ZodNumber`\>; \}, `"strip"`, `ZodTypeAny`, \{ `comparator`: `EQUAL` \| `NOT_EQUAL` \| `GREATER_THAN` \| `GREATER_THAN_OR_EQUAL` \| `LESS_THAN` \| `LESS_THAN_OR_EQUAL` \| `UNRECOGNIZED`; `required`: `boolean`; `value`: `string` \| `number`; `weight`: `number`; \}, \{ `comparator`: `EQUAL` \| `NOT_EQUAL` \| `GREATER_THAN` \| `GREATER_THAN_OR_EQUAL` \| `LESS_THAN` \| `LESS_THAN_OR_EQUAL` \| `UNRECOGNIZED`; `required`: `boolean`; `value`: `string` \| `number`; `weight`: `number`; \}\>\]\>\>

Defined in: [src/step.ts:37](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/step.ts#L37)
