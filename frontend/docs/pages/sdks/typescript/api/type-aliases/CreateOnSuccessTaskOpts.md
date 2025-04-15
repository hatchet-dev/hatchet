[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / CreateOnSuccessTaskOpts

# Type Alias: CreateOnSuccessTaskOpts\<I, O, C\>

> **CreateOnSuccessTaskOpts**\<`I`, `O`, `C`\> = `Omit`\<[`CreateBaseTaskOpts`](CreateBaseTaskOpts.md)\<`I`, `O`, `C`\>, `"name"`\>

Defined in: [src/v1/task.ts:224](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/task.ts#L224)

Options for configuring the onSuccess task that is invoked when a task succeeds.

## Type Parameters

### I

`I` *extends* [`InputType`](InputType.md) = [`UnknownInputType`](UnknownInputType.md)

The input type for the task function.

### O

`O` *extends* [`OutputType`](OutputType.md) = `void`

The return type of the task function (can be inferred from the return value of fn).

### C

`C` *extends* [`TaskFn`](TaskFn.md)\<`I`, `O`\> = [`TaskFn`](TaskFn.md)\<`I`, `O`\>
