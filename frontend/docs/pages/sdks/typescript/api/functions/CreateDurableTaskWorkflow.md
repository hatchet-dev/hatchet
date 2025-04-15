[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / CreateDurableTaskWorkflow

# Function: CreateDurableTaskWorkflow()

> **CreateDurableTaskWorkflow**\<`Fn`, `I`, `O`\>(`options`, `client?`): [`TaskWorkflowDeclaration`](../classes/TaskWorkflowDeclaration.md)\<`I`, `O`\>

Defined in: [src/v1/declaration.ts:749](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L749)

Creates a new durable task workflow declaration with types inferred from the function parameter.

## Type Parameters

### Fn

`Fn` *extends* (`input`, `ctx`) => `O` \| `Promise`\<`O`\>

The type of the durable task function

### I

`I` *extends* [`JsonObject`](../type-aliases/JsonObject.md) = `Parameters`\<`Fn`\>\[`0`\]

### O

`O` *extends* [`JsonObject`](../type-aliases/JsonObject.md) = `ReturnType`\<`Fn`\> *extends* `Promise`\<`P`\> ? `P` *extends* [`JsonObject`](../type-aliases/JsonObject.md) ? `P`\<`P`\> : `never` : `ReturnType`\<`Fn`\> *extends* [`JsonObject`](../type-aliases/JsonObject.md) ? `ReturnType`\<`ReturnType`\<`Fn`\>\> : `never`

## Parameters

### options

`object` & `Omit`\<[`CreateWorkflowDurableTaskOpts`](../type-aliases/CreateWorkflowDurableTaskOpts.md)\<`I`, `O`\>, `"fn"`\>

The durable task configuration options.

### client?

`IHatchetClient`

Optional Hatchet client instance.

## Returns

[`TaskWorkflowDeclaration`](../classes/TaskWorkflowDeclaration.md)\<`I`, `O`\>

A new TaskWorkflowDeclaration with inferred types.
