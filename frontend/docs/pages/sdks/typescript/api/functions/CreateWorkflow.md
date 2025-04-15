[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / CreateWorkflow

# Function: CreateWorkflow()

> **CreateWorkflow**\<`I`, `O`\>(`options`, `client?`): [`WorkflowDeclaration`](../classes/WorkflowDeclaration.md)\<`I`, `O`\>

Defined in: [src/v1/declaration.ts:735](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L735)

Creates a new workflow instance.

## Type Parameters

### I

`I` *extends* [`JsonObject`](../type-aliases/JsonObject.md) = [`UnknownInputType`](../type-aliases/UnknownInputType.md)

The input type for the workflow.

### O

`O` *extends* [`OutputType`](../type-aliases/OutputType.md) = `void`

The return type of the workflow.

## Parameters

### options

[`CreateWorkflowOpts`](../type-aliases/CreateWorkflowOpts.md)

The options for creating the workflow.

### client?

`IHatchetClient`

Optional Hatchet client instance.

## Returns

[`WorkflowDeclaration`](../classes/WorkflowDeclaration.md)\<`I`, `O`\>

A new Workflow instance.
