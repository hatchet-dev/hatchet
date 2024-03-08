[@hatchet-dev/typescript-sdk](../README.md) / [Exports](../modules.md) / [APIContracts](../modules/APIContracts.md) / WorkflowRun

# Interface: WorkflowRun

[APIContracts](../modules/APIContracts.md).WorkflowRun

## Table of contents

### Properties

- [displayName](APIContracts.WorkflowRun.md#displayname)
- [error](APIContracts.WorkflowRun.md#error)
- [finishedAt](APIContracts.WorkflowRun.md#finishedat)
- [input](APIContracts.WorkflowRun.md#input)
- [jobRuns](APIContracts.WorkflowRun.md#jobruns)
- [metadata](APIContracts.WorkflowRun.md#metadata)
- [startedAt](APIContracts.WorkflowRun.md#startedat)
- [status](APIContracts.WorkflowRun.md#status)
- [tenantId](APIContracts.WorkflowRun.md#tenantid)
- [triggeredBy](APIContracts.WorkflowRun.md#triggeredby)
- [workflowVersion](APIContracts.WorkflowRun.md#workflowversion)
- [workflowVersionId](APIContracts.WorkflowRun.md#workflowversionid)

## Properties

### displayName

• `Optional` **displayName**: `string`

#### Defined in

[src/clients/rest/generated/data-contracts.ts:414](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L414)

___

### error

• `Optional` **error**: `string`

#### Defined in

[src/clients/rest/generated/data-contracts.ts:418](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L418)

___

### finishedAt

• `Optional` **finishedAt**: `string`

**`Format`**

date-time

#### Defined in

[src/clients/rest/generated/data-contracts.ts:422](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L422)

___

### input

• `Optional` **input**: `Record`\<`string`, `any`\>

#### Defined in

[src/clients/rest/generated/data-contracts.ts:417](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L417)

___

### jobRuns

• `Optional` **jobRuns**: [`JobRun`](APIContracts.JobRun.md)[]

#### Defined in

[src/clients/rest/generated/data-contracts.ts:415](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L415)

___

### metadata

• **metadata**: [`APIResourceMeta`](APIContracts.APIResourceMeta.md)

#### Defined in

[src/clients/rest/generated/data-contracts.ts:409](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L409)

___

### startedAt

• `Optional` **startedAt**: `string`

**`Format`**

date-time

#### Defined in

[src/clients/rest/generated/data-contracts.ts:420](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L420)

___

### status

• **status**: [`WorkflowRunStatus`](../enums/APIContracts.WorkflowRunStatus.md)

#### Defined in

[src/clients/rest/generated/data-contracts.ts:413](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L413)

___

### tenantId

• **tenantId**: `string`

#### Defined in

[src/clients/rest/generated/data-contracts.ts:410](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L410)

___

### triggeredBy

• **triggeredBy**: [`WorkflowRunTriggeredBy`](APIContracts.WorkflowRunTriggeredBy.md)

#### Defined in

[src/clients/rest/generated/data-contracts.ts:416](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L416)

___

### workflowVersion

• `Optional` **workflowVersion**: [`WorkflowVersion`](APIContracts.WorkflowVersion.md)

#### Defined in

[src/clients/rest/generated/data-contracts.ts:412](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L412)

___

### workflowVersionId

• **workflowVersionId**: `string`

#### Defined in

[src/clients/rest/generated/data-contracts.ts:411](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L411)
