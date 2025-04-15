[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / TaskFn

# Type Alias: TaskFn()\<I, O, C\>

> **TaskFn**\<`I`, `O`, `C`\> = (`input`, `ctx`) => `O` \| `Promise`\<`O`\>

Defined in: [src/v1/task.ts:50](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/task.ts#L50)

## Type Parameters

### I

`I` *extends* [`InputType`](InputType.md) = [`UnknownInputType`](UnknownInputType.md)

### O

`O` *extends* [`OutputType`](OutputType.md) = `void`

### C

`C` = [`Context`](../classes/Context.md)\<`I`\>

## Parameters

### input

`I`

### ctx

`C`

## Returns

`O` \| `Promise`\<`O`\>
