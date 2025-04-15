[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / TaskDefaults

# Type Alias: TaskDefaults

> **TaskDefaults** = `object`

Defined in: [src/v1/declaration.ts:143](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L143)

Default configuration for all tasks in the workflow.
Can be overridden by task-specific options.

## Properties

### backoff?

> `optional` **backoff**: [`CreateWorkflowTaskOpts`](CreateWorkflowTaskOpts.md)\<`any`, `any`\>\[`"backoff"`\]

Defined in: [src/v1/declaration.ts:172](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L172)

(optional) backoff strategy configuration for retries.
- factor: Base of the exponential backoff (base ^ retry count)
- maxSeconds: Maximum backoff duration in seconds

***

### concurrency?

> `optional` **concurrency**: [`Concurrency`](Concurrency.md) \| [`Concurrency`](Concurrency.md)[]

Defined in: [src/v1/declaration.ts:192](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L192)

(optional) the concurrency options for the task.

***

### executionTimeout?

> `optional` **executionTimeout**: [`Duration`](Duration.md)

Defined in: [src/v1/declaration.ts:150](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L150)

(optional) execution timeout duration for the task after it starts running
go duration format (e.g., "1s", "5m", "1h").

default: 60s

***

### rateLimits?

> `optional` **rateLimits**: [`CreateWorkflowTaskOpts`](CreateWorkflowTaskOpts.md)\<`any`, `any`\>\[`"rateLimits"`\]

Defined in: [src/v1/declaration.ts:177](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L177)

(optional) rate limits for the task.

***

### retries?

> `optional` **retries**: [`CreateWorkflowTaskOpts`](CreateWorkflowTaskOpts.md)\<`any`, `any`\>\[`"retries"`\]

Defined in: [src/v1/declaration.ts:165](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L165)

(optional) number of retries for the task.

default: 0

***

### scheduleTimeout?

> `optional` **scheduleTimeout**: [`Duration`](Duration.md)

Defined in: [src/v1/declaration.ts:158](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L158)

(optional) schedule timeout for the task (max duration to allow the task to wait in the queue)
go duration format (e.g., "1s", "5m", "1h").

default: 5m

***

### workerLabels?

> `optional` **workerLabels**: [`CreateWorkflowTaskOpts`](CreateWorkflowTaskOpts.md)\<`any`, `any`\>\[`"desiredWorkerLabels"`\]

Defined in: [src/v1/declaration.ts:187](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L187)

(optional) worker labels for task routing and scheduling.
Each label can be a simple string/number value or an object with additional configuration:
- value: The label value (string or number)
- required: Whether the label is required for worker matching
- weight: Priority weight for worker selection
- comparator: Custom comparison logic for label matching
