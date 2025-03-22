[@hatchet-dev/typescript-sdk](../README.md) / [Exports](../modules.md) / [APIContracts](../modules/APIContracts.md) / JobRun

# Interface: JobRun

[APIContracts](../modules/APIContracts.md).JobRun

## Table of contents

### Properties

- [cancelledAt](APIContracts.JobRun.md#cancelledat)
- [cancelledError](APIContracts.JobRun.md#cancellederror)
- [cancelledReason](APIContracts.JobRun.md#cancelledreason)
- [finishedAt](APIContracts.JobRun.md#finishedat)
- [job](APIContracts.JobRun.md#job)
- [jobId](APIContracts.JobRun.md#jobid)
- [metadata](APIContracts.JobRun.md#metadata)
- [result](APIContracts.JobRun.md#result)
- [startedAt](APIContracts.JobRun.md#startedat)
- [status](APIContracts.JobRun.md#status)
- [stepRuns](APIContracts.JobRun.md#stepruns)
- [tenantId](APIContracts.JobRun.md#tenantid)
- [tickerId](APIContracts.JobRun.md#tickerid)
- [timeoutAt](APIContracts.JobRun.md#timeoutat)
- [workflowRun](APIContracts.JobRun.md#workflowrun)
- [workflowRunId](APIContracts.JobRun.md#workflowrunid)

## Properties

### cancelledAt

• `Optional` **cancelledAt**: `string`

**`Format`**

date-time

#### Defined in

[src/clients/rest/generated/data-contracts.ts:474](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L474)

___

### cancelledError

• `Optional` **cancelledError**: `string`

#### Defined in

[src/clients/rest/generated/data-contracts.ts:476](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L476)

___

### cancelledReason

• `Optional` **cancelledReason**: `string`

#### Defined in

[src/clients/rest/generated/data-contracts.ts:475](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L475)

___

### finishedAt

• `Optional` **finishedAt**: `string`

**`Format`**

date-time

#### Defined in

[src/clients/rest/generated/data-contracts.ts:470](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L470)

___

### job

• `Optional` **job**: [`Job`](APIContracts.Job.md)

#### Defined in

[src/clients/rest/generated/data-contracts.ts:462](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L462)

___

### jobId

• **jobId**: `string`

#### Defined in

[src/clients/rest/generated/data-contracts.ts:461](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L461)

___

### metadata

• **metadata**: [`APIResourceMeta`](APIContracts.APIResourceMeta.md)

#### Defined in

[src/clients/rest/generated/data-contracts.ts:457](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L457)

___

### result

• `Optional` **result**: `object`

#### Defined in

[src/clients/rest/generated/data-contracts.ts:466](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L466)

___

### startedAt

• `Optional` **startedAt**: `string`

**`Format`**

date-time

#### Defined in

[src/clients/rest/generated/data-contracts.ts:468](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L468)

___

### status

• **status**: [`JobRunStatus`](../enums/APIContracts.JobRunStatus.md)

#### Defined in

[src/clients/rest/generated/data-contracts.ts:465](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L465)

___

### stepRuns

• `Optional` **stepRuns**: [`StepRun`](APIContracts.StepRun.md)[]

#### Defined in

[src/clients/rest/generated/data-contracts.ts:464](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L464)

___

### tenantId

• **tenantId**: `string`

#### Defined in

[src/clients/rest/generated/data-contracts.ts:458](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L458)

___

### tickerId

• `Optional` **tickerId**: `string`

#### Defined in

[src/clients/rest/generated/data-contracts.ts:463](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L463)

___

### timeoutAt

• `Optional` **timeoutAt**: `string`

**`Format`**

date-time

#### Defined in

[src/clients/rest/generated/data-contracts.ts:472](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L472)

___

### workflowRun

• `Optional` **workflowRun**: [`WorkflowRun`](APIContracts.WorkflowRun.md)

#### Defined in

[src/clients/rest/generated/data-contracts.ts:460](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L460)

___

### workflowRunId

• **workflowRunId**: `string`

#### Defined in

[src/clients/rest/generated/data-contracts.ts:459](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L459)
