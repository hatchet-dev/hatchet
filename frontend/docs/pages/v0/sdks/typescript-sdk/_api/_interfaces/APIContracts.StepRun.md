[@hatchet-dev/typescript-sdk](../README.md) / [Exports](../modules.md) / [APIContracts](../modules/APIContracts.md) / StepRun

# Interface: StepRun

[APIContracts](../modules/APIContracts.md).StepRun

## Table of contents

### Properties

- [cancelledAt](APIContracts.StepRun.md#cancelledat)
- [cancelledAtEpoch](APIContracts.StepRun.md#cancelledatepoch)
- [cancelledError](APIContracts.StepRun.md#cancellederror)
- [cancelledReason](APIContracts.StepRun.md#cancelledreason)
- [children](APIContracts.StepRun.md#children)
- [error](APIContracts.StepRun.md#error)
- [finishedAt](APIContracts.StepRun.md#finishedat)
- [finishedAtEpoch](APIContracts.StepRun.md#finishedatepoch)
- [input](APIContracts.StepRun.md#input)
- [jobRun](APIContracts.StepRun.md#jobrun)
- [jobRunId](APIContracts.StepRun.md#jobrunid)
- [metadata](APIContracts.StepRun.md#metadata)
- [output](APIContracts.StepRun.md#output)
- [parents](APIContracts.StepRun.md#parents)
- [requeueAfter](APIContracts.StepRun.md#requeueafter)
- [result](APIContracts.StepRun.md#result)
- [startedAt](APIContracts.StepRun.md#startedat)
- [startedAtEpoch](APIContracts.StepRun.md#startedatepoch)
- [status](APIContracts.StepRun.md#status)
- [step](APIContracts.StepRun.md#step)
- [stepId](APIContracts.StepRun.md#stepid)
- [tenantId](APIContracts.StepRun.md#tenantid)
- [timeoutAt](APIContracts.StepRun.md#timeoutat)
- [timeoutAtEpoch](APIContracts.StepRun.md#timeoutatepoch)
- [workerId](APIContracts.StepRun.md#workerid)

## Properties

### cancelledAt

• `Optional` **cancelledAt**: `string`

**`Format`**

date-time

#### Defined in

[src/clients/rest/generated/data-contracts.ts:515](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L515)

___

### cancelledAtEpoch

• `Optional` **cancelledAtEpoch**: `number`

#### Defined in

[src/clients/rest/generated/data-contracts.ts:516](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L516)

___

### cancelledError

• `Optional` **cancelledError**: `string`

#### Defined in

[src/clients/rest/generated/data-contracts.ts:518](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L518)

___

### cancelledReason

• `Optional` **cancelledReason**: `string`

#### Defined in

[src/clients/rest/generated/data-contracts.ts:517](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L517)

___

### children

• `Optional` **children**: `string`[]

#### Defined in

[src/clients/rest/generated/data-contracts.ts:495](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L495)

___

### error

• `Optional` **error**: `string`

#### Defined in

[src/clients/rest/generated/data-contracts.ts:504](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L504)

___

### finishedAt

• `Optional` **finishedAt**: `string`

**`Format`**

date-time

#### Defined in

[src/clients/rest/generated/data-contracts.ts:509](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L509)

___

### finishedAtEpoch

• `Optional` **finishedAtEpoch**: `number`

#### Defined in

[src/clients/rest/generated/data-contracts.ts:510](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L510)

___

### input

• `Optional` **input**: `string`

#### Defined in

[src/clients/rest/generated/data-contracts.ts:498](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L498)

___

### jobRun

• `Optional` **jobRun**: [`JobRun`](APIContracts.JobRun.md)

#### Defined in

[src/clients/rest/generated/data-contracts.ts:492](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L492)

___

### jobRunId

• **jobRunId**: `string`

#### Defined in

[src/clients/rest/generated/data-contracts.ts:491](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L491)

___

### metadata

• **metadata**: [`APIResourceMeta`](APIContracts.APIResourceMeta.md)

#### Defined in

[src/clients/rest/generated/data-contracts.ts:489](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L489)

___

### output

• `Optional` **output**: `string`

#### Defined in

[src/clients/rest/generated/data-contracts.ts:499](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L499)

___

### parents

• `Optional` **parents**: `string`[]

#### Defined in

[src/clients/rest/generated/data-contracts.ts:496](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L496)

___

### requeueAfter

• `Optional` **requeueAfter**: `string`

**`Format`**

date-time

#### Defined in

[src/clients/rest/generated/data-contracts.ts:502](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L502)

___

### result

• `Optional` **result**: `object`

#### Defined in

[src/clients/rest/generated/data-contracts.ts:503](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L503)

___

### startedAt

• `Optional` **startedAt**: `string`

**`Format`**

date-time

#### Defined in

[src/clients/rest/generated/data-contracts.ts:506](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L506)

___

### startedAtEpoch

• `Optional` **startedAtEpoch**: `number`

#### Defined in

[src/clients/rest/generated/data-contracts.ts:507](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L507)

___

### status

• **status**: [`StepRunStatus`](../enums/APIContracts.StepRunStatus.md)

#### Defined in

[src/clients/rest/generated/data-contracts.ts:500](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L500)

___

### step

• `Optional` **step**: [`Step`](APIContracts.Step.md)

#### Defined in

[src/clients/rest/generated/data-contracts.ts:494](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L494)

___

### stepId

• **stepId**: `string`

#### Defined in

[src/clients/rest/generated/data-contracts.ts:493](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L493)

___

### tenantId

• **tenantId**: `string`

#### Defined in

[src/clients/rest/generated/data-contracts.ts:490](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L490)

___

### timeoutAt

• `Optional` **timeoutAt**: `string`

**`Format`**

date-time

#### Defined in

[src/clients/rest/generated/data-contracts.ts:512](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L512)

___

### timeoutAtEpoch

• `Optional` **timeoutAtEpoch**: `number`

#### Defined in

[src/clients/rest/generated/data-contracts.ts:513](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L513)

___

### workerId

• `Optional` **workerId**: `string`

#### Defined in

[src/clients/rest/generated/data-contracts.ts:497](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L497)
