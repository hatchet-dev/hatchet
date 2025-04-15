[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / WorkflowConcurrency

# Variable: WorkflowConcurrency

> `const` **WorkflowConcurrency**: `ZodObject`\<\{ `expression`: `ZodOptional`\<`ZodString`\>; `limitStrategy`: `ZodOptional`\<`ZodNativeEnum`\<*typeof* `ConcurrencyLimitStrategy`\>\>; `maxRuns`: `ZodOptional`\<`ZodNumber`\>; `name`: `ZodString`; \}, `"strip"`, `ZodTypeAny`, \{ `expression`: `string`; `limitStrategy`: `CANCEL_IN_PROGRESS` \| `DROP_NEWEST` \| `QUEUE_NEWEST` \| `GROUP_ROUND_ROBIN` \| `CANCEL_NEWEST` \| `UNRECOGNIZED`; `maxRuns`: `number`; `name`: `string`; \}, \{ `expression`: `string`; `limitStrategy`: `CANCEL_IN_PROGRESS` \| `DROP_NEWEST` \| `QUEUE_NEWEST` \| `GROUP_ROUND_ROBIN` \| `CANCEL_NEWEST` \| `UNRECOGNIZED`; `maxRuns`: `number`; `name`: `string`; \}\>

Defined in: [src/workflow.ts:27](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/workflow.ts#L27)
