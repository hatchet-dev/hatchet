[@hatchet-dev/typescript-sdk](../README.md) / [Exports](../modules.md) / CreateStep

# Interface: CreateStep\<T, K\>

## Type parameters

| Name |
| :------ |
| `T` |
| `K` |

## Hierarchy

- `TypeOf`\<typeof [`CreateStepSchema`](../modules.md#createstepschema)\>

  ↳ **`CreateStep`**

## Table of contents

### Properties

- [name](CreateStep.md#name)
- [parents](CreateStep.md#parents)
- [run](CreateStep.md#run)
- [timeout](CreateStep.md#timeout)

## Properties

### name

• **name**: `string`

#### Inherited from

z.infer.name

#### Defined in

[src/step.ts:6](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/step.ts#L6)

[src/step.ts:6](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/step.ts#L6)

___

### parents

• `Optional` **parents**: `string`[]

#### Inherited from

z.infer.parents

#### Defined in

[src/step.ts:7](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/step.ts#L7)

___

### run

• **run**: [`StepRunFunction`](../modules.md#steprunfunction)\<`T`, `K`\>

#### Defined in

[src/step.ts:58](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/step.ts#L58)

___

### timeout

• `Optional` **timeout**: `string`

#### Inherited from

z.infer.timeout

#### Defined in

[src/step.ts:8](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/step.ts#L8)
