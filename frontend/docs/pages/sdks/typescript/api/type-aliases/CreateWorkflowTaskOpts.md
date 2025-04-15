[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / CreateWorkflowTaskOpts

# Type Alias: CreateWorkflowTaskOpts\<I, O, C\>

> **CreateWorkflowTaskOpts**\<`I`, `O`, `C`\> = [`CreateBaseTaskOpts`](CreateBaseTaskOpts.md)\<`I`, `O`, `C`\> & `object`

Defined in: [src/v1/task.ts:141](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/task.ts#L141)

## Type declaration

### cancelIf?

> `optional` **cancelIf**: [`Conditions`](Conditions.md) \| [`Conditions`](Conditions.md)[]

(optional) cancel the task if the conditions are met
all provided conditions must be met (AND logic)
use Or() to create a condition that waits for any of the provided conditions to be met (OR logic)

#### Examples

```
cancelIf: { eventKey: 'user:update' } // cancel the task if the user:update event is received
```

```
cancelIf: [{ sleepFor: 5 }, Or({ eventKey: 'user:update' }, { eventKey: 'user:delete' })] // cancel the task if the sleep or both user:update or user:delete are met

### parents?

> `optional` **parents**: `CreateWorkflowTaskOpts`\<`I`, `any`, `any`\>[]

Parent tasks that must complete before this task runs.
Used to define the directed acyclic graph (DAG) of the workflow.

### skipIf?

> `optional` **skipIf**: [`Conditions`](Conditions.md) \| [`Conditions`](Conditions.md)[]

(optional) skip the task if the conditions are met
all provided conditions must be met (AND logic)
use Or() to create a condition that waits for any of the provided conditions to be met (OR logic)

#### Examples

```
skipIf: [{ eventKey: 'user:update' }] // skip the task if the user:update event is received
```

```
skipIf: [{ sleepFor: 5 }, Or({ eventKey: 'user:update' }, { eventKey: 'user:delete' })] // skip the task if the sleep or both user:update or user:delete are met
```

```
skipIf: [{ parent: firstTask }] // skip the task if the parent task completes
```

### waitFor?

> `optional` **waitFor**: [`Conditions`](Conditions.md) \| [`Conditions`](Conditions.md)[]

(optional) the conditions to match before the task is queued
all provided conditions must be met (AND logic)
use Or() to create a condition that waits for any of the provided conditions to be met (OR logic)

#### Examples

```
waitFor: [{ sleepFor: 5 }, { eventKey: 'user:update' }] // all conditions must be met
```

```
waitFor: Or({ eventKey: 'user:update' }, { parent: firstTask }) // any of the conditions must be met
```

```
waitFor: [{ sleepFor: 5 }, Or({ eventKey: 'user:update' }, { eventKey: 'user:delete' })] // sleep or both user:update or user:delete must be met
```

## Type Parameters

### I

`I` *extends* [`InputType`](InputType.md) = [`UnknownInputType`](UnknownInputType.md)

### O

`O` *extends* [`OutputType`](OutputType.md) = `void`

### C

`C` *extends* [`TaskFn`](TaskFn.md)\<`I`, `O`\> \| [`DurableTaskFn`](DurableTaskFn.md)\<`I`, `O`\> = [`TaskFn`](TaskFn.md)\<`I`, `O`\>
