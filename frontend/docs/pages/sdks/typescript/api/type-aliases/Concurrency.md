[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / Concurrency

# Type Alias: Concurrency

> **Concurrency** = `object`

Defined in: [src/v1/task.ts:10](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/task.ts#L10)

Options for configuring the concurrency for a task.

## Properties

### expression

> **expression**: `string`

Defined in: [src/v1/task.ts:19](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/task.ts#L19)

required the CEL expression to use for concurrency

#### Example

```
"input.key" // use the value of the key in the input
```

***

### limitStrategy?

> `optional` **limitStrategy**: `ConcurrencyLimitStrategy`

Defined in: [src/v1/task.ts:33](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/task.ts#L33)

(optional) the strategy to use when the concurrency limit is reached

default: CANCEL_IN_PROGRESS

***

### maxRuns?

> `optional` **maxRuns**: `number`

Defined in: [src/v1/task.ts:26](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/task.ts#L26)

(optional) the maximum number of concurrent workflow runs

default: 1
