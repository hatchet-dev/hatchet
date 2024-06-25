[@hatchet-dev/typescript-sdk](../README.md) / [Exports](../modules.md) / [APIContracts](../modules/APIContracts.md) / EventWorkflowRunSummary

# Interface: EventWorkflowRunSummary

[APIContracts](../modules/APIContracts.md).EventWorkflowRunSummary

## Table of contents

### Properties

- [failed](APIContracts.EventWorkflowRunSummary.md#failed)
- [pending](APIContracts.EventWorkflowRunSummary.md#pending)
- [running](APIContracts.EventWorkflowRunSummary.md#running)
- [succeeded](APIContracts.EventWorkflowRunSummary.md#succeeded)

## Properties

### failed

• `Optional` **failed**: `number`

The number of failed runs.

**`Format`**

int64

#### Defined in

[src/clients/rest/generated/data-contracts.ts:277](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L277)

___

### pending

• `Optional` **pending**: `number`

The number of pending runs.

**`Format`**

int64

#### Defined in

[src/clients/rest/generated/data-contracts.ts:262](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L262)

___

### running

• `Optional` **running**: `number`

The number of running runs.

**`Format`**

int64

#### Defined in

[src/clients/rest/generated/data-contracts.ts:267](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L267)

___

### succeeded

• `Optional` **succeeded**: `number`

The number of succeeded runs.

**`Format`**

int64

#### Defined in

[src/clients/rest/generated/data-contracts.ts:272](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L272)
