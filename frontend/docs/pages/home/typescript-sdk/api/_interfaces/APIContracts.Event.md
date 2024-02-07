[@hatchet-dev/typescript-sdk](../README.md) / [Exports](../modules.md) / [APIContracts](../modules/APIContracts.md) / Event

# Interface: Event

[APIContracts](../modules/APIContracts.md).Event

## Table of contents

### Properties

- [key](APIContracts.Event.md#key)
- [metadata](APIContracts.Event.md#metadata)
- [tenant](APIContracts.Event.md#tenant)
- [tenantId](APIContracts.Event.md#tenantid)
- [workflowRunSummary](APIContracts.Event.md#workflowrunsummary)

## Properties

### key

• **key**: `string`

The key for the event.

#### Defined in

[src/clients/rest/generated/data-contracts.ts:243](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L243)

___

### metadata

• **metadata**: [`APIResourceMeta`](APIContracts.APIResourceMeta.md)

#### Defined in

[src/clients/rest/generated/data-contracts.ts:241](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L241)

___

### tenant

• `Optional` **tenant**: [`Tenant`](APIContracts.Tenant.md)

The tenant associated with this event.

#### Defined in

[src/clients/rest/generated/data-contracts.ts:245](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L245)

___

### tenantId

• **tenantId**: `string`

The ID of the tenant associated with this event.

#### Defined in

[src/clients/rest/generated/data-contracts.ts:247](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L247)

___

### workflowRunSummary

• `Optional` **workflowRunSummary**: [`EventWorkflowRunSummary`](APIContracts.EventWorkflowRunSummary.md)

The workflow run summary for this event.

#### Defined in

[src/clients/rest/generated/data-contracts.ts:249](https://github.com/hatchet-dev/hatchet/blob/af21f67/typescript-sdk/src/clients/rest/generated/data-contracts.ts#L249)
