[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / TaskOutput

# Type Alias: TaskOutput\<O, Key, Fallback\>

> **TaskOutput**\<`O`, `Key`, `Fallback`\> = `O` *extends* `Record`\<`Key`, infer Value\> ? `Value` *extends* [`OutputType`](OutputType.md) ? `Value` : `Fallback` : `Fallback`

Defined in: [src/v1/declaration.ts:57](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L57)

Helper type to safely extract output types from task results

## Type Parameters

### O

`O`

### Key

`Key` *extends* `string`

### Fallback

`Fallback`
