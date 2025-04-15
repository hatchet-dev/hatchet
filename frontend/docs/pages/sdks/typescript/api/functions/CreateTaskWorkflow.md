[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / CreateTaskWorkflow

# Function: CreateTaskWorkflow()

> **CreateTaskWorkflow**\<`Fn`, `I`, `O`\>(`options`, `client?`): [`TaskWorkflowDeclaration`](../classes/TaskWorkflowDeclaration.md)\<`I`, `O`\>

Defined in: [src/v1/declaration.ts:708](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L708)

Creates a new task workflow declaration with types inferred from the function parameter.

## Type Parameters

### Fn

`Fn` *extends* (`input`, `ctx?`) => `O` \| `Promise`\<`O`\>

The type of the task function

### I

`I` *extends* [`JsonObject`](../type-aliases/JsonObject.md) = `Parameters`\<`Fn`\>\[`0`\]

### O

`O` *extends* [`OutputType`](../type-aliases/OutputType.md) = `ReturnType`\<`Fn`\> *extends* `Promise`\<`P`\> ? `P` *extends* [`OutputType`](../type-aliases/OutputType.md) ? `P`\<`P`\> : `void` : `ReturnType`\<`Fn`\> *extends* [`OutputType`](../type-aliases/OutputType.md) ? `ReturnType`\<`ReturnType`\<`Fn`\>\> : `void`

## Parameters

### options

`object` & `Omit`\<[`CreateTaskWorkflowOpts`](../type-aliases/CreateTaskWorkflowOpts.md)\<`I`, `O`\>, `"fn"`\>

The task configuration options.

### client?

`IHatchetClient`

Optional Hatchet client instance.

## Returns

[`TaskWorkflowDeclaration`](../classes/TaskWorkflowDeclaration.md)\<`I`, `O`\>

A new TaskWorkflowDeclaration with inferred types.
