[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / CreateWorkflowDurableTaskOpts

# Type Alias: CreateWorkflowDurableTaskOpts\<I, O, C\>

> **CreateWorkflowDurableTaskOpts**\<`I`, `O`, `C`\> = [`CreateWorkflowTaskOpts`](CreateWorkflowTaskOpts.md)\<`I`, `O`, `C`\>

Defined in: [src/v1/task.ts:213](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/task.ts#L213)

Options for creating a hatchet durable task which is an atomic unit of work in a workflow.

## Type Parameters

### I

`I` *extends* [`InputType`](InputType.md) = [`UnknownInputType`](UnknownInputType.md)

The input type for the task function.

### O

`O` *extends* [`OutputType`](OutputType.md) = `void`

The return type of the task function (can be inferred from the return value of fn).

### C

`C` *extends* [`DurableTaskFn`](DurableTaskFn.md)\<`I`, `O`\> = [`DurableTaskFn`](DurableTaskFn.md)\<`I`, `O`\>
