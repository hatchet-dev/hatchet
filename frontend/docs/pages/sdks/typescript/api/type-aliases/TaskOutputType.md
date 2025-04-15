[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / TaskOutputType

# Type Alias: TaskOutputType\<O, TaskName, InferredType\>

> **TaskOutputType**\<`O`, `TaskName`, `InferredType`\> = `TaskName` *extends* keyof `O` ? `O`\[`TaskName`\] *extends* [`OutputType`](OutputType.md) ? `O`\[`TaskName`\] : `InferredType` : `InferredType`

Defined in: [src/v1/declaration.ts:63](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L63)

Extracts a property from an object type based on task name, or falls back to inferred type

## Type Parameters

### O

`O`

### TaskName

`TaskName` *extends* `string`

### InferredType

`InferredType` *extends* [`OutputType`](OutputType.md)
