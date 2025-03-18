[@hatchet-dev/typescript-sdk](../README.md) / [Exports](../modules.md) / [APIContracts](../modules/APIContracts.md) / Worker

# Interface: Worker

[APIContracts](../modules/APIContracts.md).Worker

## Table of contents

### Properties

- [actions](APIContracts.Worker.md#actions)
- [lastHeartbeatAt](APIContracts.Worker.md#lastheartbeatat)
- [metadata](APIContracts.Worker.md#metadata)
- [name](APIContracts.Worker.md#name)
- [recentStepRuns](APIContracts.Worker.md#recentstepruns)

## Properties

### actions

• `Optional` **actions**: `string`[]

The actions this worker can perform.

#### Defined in

[src/clients/rest/generated/data-contracts.ts:537](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L537)

___

### lastHeartbeatAt

• `Optional` **lastHeartbeatAt**: `string`

The time this worker last sent a heartbeat.

**`Format`**

date-time

**`Example`**

```ts
"2022-12-13T20:06:48.888Z"
```

#### Defined in

[src/clients/rest/generated/data-contracts.ts:535](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L535)

___

### metadata

• **metadata**: [`APIResourceMeta`](APIContracts.APIResourceMeta.md)

#### Defined in

[src/clients/rest/generated/data-contracts.ts:527](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L527)

___

### name

• **name**: `string`

The name of the worker.

#### Defined in

[src/clients/rest/generated/data-contracts.ts:529](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L529)

___

### recentStepRuns

• `Optional` **recentStepRuns**: [`StepRun`](APIContracts.StepRun.md)[]

The recent step runs for this worker.

#### Defined in

[src/clients/rest/generated/data-contracts.ts:539](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L539)
