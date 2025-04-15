[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / WorkflowsClient

# Class: WorkflowsClient

Defined in: [src/v1/client/features/workflows.ts:12](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/workflows.ts#L12)

WorkflowsClient is used to list and manage workflows

## Constructors

### Constructor

> **new WorkflowsClient**(`client`, `cacheTTL?`): `WorkflowsClient`

Defined in: [src/v1/client/features/workflows.ts:20](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/workflows.ts#L20)

#### Parameters

##### client

[`HatchetClient`](HatchetClient.md)

##### cacheTTL?

`number`

#### Returns

`WorkflowsClient`

## Properties

### api

> **api**: [`Api`](Api.md)\<`unknown`\>

Defined in: [src/v1/client/features/workflows.ts:13](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/workflows.ts#L13)

***

### tenantId

> **tenantId**: `string`

Defined in: [src/v1/client/features/workflows.ts:14](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/workflows.ts#L14)

## Methods

### delete()

> **delete**(`workflow`): `Promise`\<`void`\>

Defined in: [src/v1/client/features/workflows.ts:72](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/workflows.ts#L72)

#### Parameters

##### workflow

`string` | [`Workflow`](../interfaces/Workflow.md) | [`BaseWorkflowDeclaration`](BaseWorkflowDeclaration.md)\<`any`, `any`\>

#### Returns

`Promise`\<`void`\>

***

### get()

> **get**(`workflow`): `Promise`\<`any`\>

Defined in: [src/v1/client/features/workflows.ts:29](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/workflows.ts#L29)

#### Parameters

##### workflow

`string` | [`Workflow`](../interfaces/Workflow.md) | [`BaseWorkflowDeclaration`](BaseWorkflowDeclaration.md)\<`any`, `any`\>

#### Returns

`Promise`\<`any`\>

***

### list()

> **list**(`opts?`): `Promise`\<[`WorkflowList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowList.md)\>

Defined in: [src/v1/client/features/workflows.ts:67](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/client/features/workflows.ts#L67)

#### Parameters

##### opts?

###### limit?

`number`

The number to limit by

**Format**

int

**Default**

```ts
50
```

###### name?

`string`

Search by name

###### offset?

`number`

The number to skip

**Format**

int

**Default**

```ts
0
```

#### Returns

`Promise`\<[`WorkflowList`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WorkflowList.md)\>
