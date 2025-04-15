[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / CreateDurableTaskWorkflowOpts

# Type Alias: CreateDurableTaskWorkflowOpts\<I, O\>

> **CreateDurableTaskWorkflowOpts**\<`I`, `O`\> = [`CreateBaseWorkflowOpts`](CreateBaseWorkflowOpts.md) & [`CreateBaseTaskOpts`](CreateBaseTaskOpts.md)\<`I`, `O`, [`DurableTaskFn`](DurableTaskFn.md)\<`I`, `O`\>\>

Defined in: [src/v1/declaration.ts:124](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L124)

## Type Parameters

### I

`I` *extends* [`InputType`](InputType.md) = [`UnknownInputType`](UnknownInputType.md)

### O

`O` *extends* [`OutputType`](OutputType.md) = `void`
