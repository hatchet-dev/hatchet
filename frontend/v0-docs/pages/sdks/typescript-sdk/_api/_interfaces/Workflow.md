[@hatchet-dev/typescript-sdk](../README.md) / [Exports](../modules.md) / Workflow

# Interface: Workflow

## Hierarchy

- `TypeOf`\<typeof [`CreateWorkflowSchema`](../modules.md#createworkflowschema)\>

  ↳ **`Workflow`**

## Table of contents

### Properties

- [concurrency](Workflow.md#concurrency)
- [description](Workflow.md#description)
- [id](Workflow.md#id)
- [on](Workflow.md#on)
- [steps](Workflow.md#steps)
- [timeout](Workflow.md#timeout)
- [version](Workflow.md#version)

## Properties

### concurrency

• `Optional` **concurrency**: \{ `limitStrategy?`: `CANCEL_IN_PROGRESS` \| `DROP_NEWEST` \| `QUEUE_NEWEST` \| `UNRECOGNIZED` ; `maxRuns?`: `number` ; `name`: `string`  } & \{ `key`: (`ctx`: `any`) => `string`  }

#### Defined in

[src/workflow.ts:42](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/workflow.ts#L42)

___

### description

• **description**: `string`

#### Inherited from

z.infer.description

#### Defined in

[src/workflow.ts:34](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/workflow.ts#L34)

[src/workflow.ts:34](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/workflow.ts#L34)

___

### id

• **id**: `string`

#### Inherited from

z.infer.id

#### Defined in

[src/workflow.ts:33](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/workflow.ts#L33)

[src/workflow.ts:33](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/workflow.ts#L33)

___

### on

• **on**: \{ `cron`: `string` ; `event?`: `undefined`  } \| \{ `cron?`: `undefined` ; `event`: `string`  } & `undefined` \| \{ `cron`: `string` ; `event?`: `undefined`  } \| \{ `cron?`: `undefined` ; `event`: `string`  } = `OnConfigSchema`

#### Inherited from

z.infer.on

#### Defined in

[src/workflow.ts:37](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/workflow.ts#L37)

[src/workflow.ts:37](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/workflow.ts#L37)

___

### steps

• **steps**: [`CreateStep`](CreateStep.md)\<`any`, `any`\>[]

#### Overrides

z.infer.steps

#### Defined in

[src/workflow.ts:45](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/workflow.ts#L45)

___

### timeout

• `Optional` **timeout**: `string`

#### Inherited from

z.infer.timeout

#### Defined in

[src/workflow.ts:36](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/workflow.ts#L36)

___

### version

• `Optional` **version**: `string`

#### Inherited from

z.infer.version

#### Defined in

[src/workflow.ts:35](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/workflow.ts#L35)
