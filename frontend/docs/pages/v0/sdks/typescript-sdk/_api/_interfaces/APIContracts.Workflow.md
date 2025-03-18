[@hatchet-dev/typescript-sdk](../README.md) / [Exports](../modules.md) / [APIContracts](../modules/APIContracts.md) / Workflow

# Interface: Workflow

[APIContracts](../modules/APIContracts.md).Workflow

## Table of contents

### Properties

- [description](APIContracts.Workflow.md#description)
- [jobs](APIContracts.Workflow.md#jobs)
- [lastRun](APIContracts.Workflow.md#lastrun)
- [metadata](APIContracts.Workflow.md#metadata)
- [name](APIContracts.Workflow.md#name)
- [tags](APIContracts.Workflow.md#tags)
- [versions](APIContracts.Workflow.md#versions)

## Properties

### description

• `Optional` **description**: `string`

The description of the workflow.

#### Defined in

[src/clients/rest/generated/data-contracts.ts:316](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L316)

___

### jobs

• `Optional` **jobs**: [`Job`](APIContracts.Job.md)[]

The jobs of the workflow.

#### Defined in

[src/clients/rest/generated/data-contracts.ts:322](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L322)

___

### lastRun

• `Optional` **lastRun**: [`WorkflowRun`](APIContracts.WorkflowRun.md)

#### Defined in

[src/clients/rest/generated/data-contracts.ts:320](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L320)

___

### metadata

• **metadata**: [`APIResourceMeta`](APIContracts.APIResourceMeta.md)

#### Defined in

[src/clients/rest/generated/data-contracts.ts:312](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L312)

___

### name

• **name**: `string`

The name of the workflow.

#### Defined in

[src/clients/rest/generated/data-contracts.ts:314](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L314)

___

### tags

• `Optional` **tags**: [`WorkflowTag`](APIContracts.WorkflowTag.md)[]

The tags of the workflow.

#### Defined in

[src/clients/rest/generated/data-contracts.ts:319](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L319)

___

### versions

• `Optional` **versions**: [`WorkflowVersionMeta`](APIContracts.WorkflowVersionMeta.md)[]

#### Defined in

[src/clients/rest/generated/data-contracts.ts:317](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L317)
