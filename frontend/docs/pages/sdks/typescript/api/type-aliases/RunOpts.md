[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / RunOpts

# Type Alias: RunOpts

> **RunOpts** = `object`

Defined in: [src/v1/declaration.ts:40](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L40)

Options for running a workflow.

## Properties

### additionalMetadata?

> `optional` **additionalMetadata**: `AdditionalMetadata`

Defined in: [src/v1/declaration.ts:44](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L44)

(optional) additional metadata to attach to the workflow run.

***

### priority?

> `optional` **priority**: [`Priority`](../enumerations/Priority.md)

Defined in: [src/v1/declaration.ts:51](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L51)

(optional) the priority for the workflow run.

values: Priority.LOW, Priority.MEDIUM, Priority.HIGH (1, 2, or 3 )
