[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / CreateBaseTaskOpts

# Type Alias: CreateBaseTaskOpts\<I, O, C\>

> **CreateBaseTaskOpts**\<`I`, `O`, `C`\> = `object`

Defined in: [src/v1/task.ts:67](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/task.ts#L67)

Options for creating a hatchet task which is an atomic unit of work in a workflow.

## Type Parameters

### I

`I` *extends* [`InputType`](InputType.md) = [`UnknownInputType`](UnknownInputType.md)

The input type for the task function.

### O

`O` *extends* [`OutputType`](OutputType.md) = `void`

The return type of the task function (can be inferred from the return value of fn).

### C

`C` = [`TaskFn`](TaskFn.md)\<`I`, `O`\>

## Properties

### backoff?

> `optional` **backoff**: [`CreateStep`](../interfaces/CreateStep.md)\<`I`, `O`\>\[`"backoff"`\]

Defined in: [src/v1/task.ts:118](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/task.ts#L118)

(optional) backoff strategy configuration for retries.
- factor: Base of the exponential backoff (base ^ retry count)
- maxSeconds: Maximum backoff duration in seconds

***

### concurrency?

> `optional` **concurrency**: [`Concurrency`](Concurrency.md) \| [`Concurrency`](Concurrency.md)[]

Defined in: [src/v1/task.ts:138](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/task.ts#L138)

(optional) the concurrency options for the task

***

### desiredWorkerLabels?

> `optional` **desiredWorkerLabels**: [`CreateStep`](../interfaces/CreateStep.md)\<`I`, `O`\>\[`"worker_labels"`\]

Defined in: [src/v1/task.ts:133](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/task.ts#L133)

(optional) worker labels for task routing and scheduling.
Each label can be a simple string/number value or an object with additional configuration:
- value: The label value (string or number)
- required: Whether the label is required for worker matching
- weight: Priority weight for worker selection
- comparator: Custom comparison logic for label matching

***

### executionTimeout?

> `optional` **executionTimeout**: [`Duration`](Duration.md)

Defined in: [src/v1/task.ts:96](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/task.ts#L96)

(optional) execution timeout duration for the task after it starts running
go duration format (e.g., "1s", "5m", "1h").

default: 60s

***

### fn

> **fn**: `C`

Defined in: [src/v1/task.ts:83](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/task.ts#L83)

The function to execute when the task runs.

#### Param

The input data for the workflow invocation.

#### Param

The execution context for the task.

#### Returns

The result of the task execution.

***

### name

> **name**: `string`

Defined in: [src/v1/task.ts:75](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/task.ts#L75)

The name of the task.

***

### rateLimits?

> `optional` **rateLimits**: [`CreateStep`](../interfaces/CreateStep.md)\<`I`, `O`\>\[`"rate_limits"`\]

Defined in: [src/v1/task.ts:123](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/task.ts#L123)

(optional) rate limits for the task.

***

### retries?

> `optional` **retries**: [`CreateStep`](../interfaces/CreateStep.md)\<`I`, `O`\>\[`"retries"`\]

Defined in: [src/v1/task.ts:111](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/task.ts#L111)

(optional) number of retries for the task.

default: 0

***

### scheduleTimeout?

> `optional` **scheduleTimeout**: [`Duration`](Duration.md)

Defined in: [src/v1/task.ts:104](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/task.ts#L104)

(optional) schedule timeout for the task (max duration to allow the task to wait in the queue)
go duration format (e.g., "1s", "5m", "1h").

default: 5m

***

### ~~timeout?~~

> `optional` **timeout**: [`CreateStep`](../interfaces/CreateStep.md)\<`I`, `O`\>\[`"timeout"`\]

Defined in: [src/v1/task.ts:88](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/task.ts#L88)

#### Deprecated

use executionTimeout instead
