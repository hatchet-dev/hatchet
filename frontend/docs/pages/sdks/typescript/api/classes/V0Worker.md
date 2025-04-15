[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / V0Worker

# Class: V0Worker

Defined in: [src/clients/worker/worker.ts:47](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/worker/worker.ts#L47)

## Constructors

### Constructor

> **new V0Worker**(`client`, `options`): `V0Worker`

Defined in: [src/clients/worker/worker.ts:67](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/worker/worker.ts#L67)

#### Parameters

##### client

`InternalHatchetClient`

##### options

###### handleKill?

`boolean`

###### labels?

`WorkerLabels`

###### maxRuns?

`number`

###### name

`string`

#### Returns

`V0Worker`

## Properties

### action\_registry

> **action\_registry**: [`ActionRegistry`](../type-aliases/ActionRegistry.md)

Defined in: [src/clients/worker/worker.ts:54](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/worker/worker.ts#L54)

***

### client

> **client**: `InternalHatchetClient`

Defined in: [src/clients/worker/worker.ts:48](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/worker/worker.ts#L48)

***

### contexts

> **contexts**: `Record`\<`string`, [`Context`](Context.md)\<`any`, `any`\>\> = `{}`

Defined in: [src/clients/worker/worker.ts:58](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/worker/worker.ts#L58)

***

### futures

> **futures**: `Record`\<`string`, `HatchetPromise`\<`any`\>\> = `{}`

Defined in: [src/clients/worker/worker.ts:57](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/worker/worker.ts#L57)

***

### handle\_kill

> **handle\_kill**: `boolean`

Defined in: [src/clients/worker/worker.ts:52](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/worker/worker.ts#L52)

***

### killing

> **killing**: `boolean`

Defined in: [src/clients/worker/worker.ts:51](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/worker/worker.ts#L51)

***

### labels

> **labels**: `WorkerLabels` = `{}`

Defined in: [src/clients/worker/worker.ts:65](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/worker/worker.ts#L65)

***

### listener

> **listener**: `undefined` \| `ActionListener`

Defined in: [src/clients/worker/worker.ts:56](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/worker/worker.ts#L56)

***

### logger

> **logger**: `Logger`

Defined in: [src/clients/worker/worker.ts:61](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/worker/worker.ts#L61)

***

### maxRuns?

> `optional` **maxRuns**: `number`

Defined in: [src/clients/worker/worker.ts:59](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/worker/worker.ts#L59)

***

### name

> **name**: `string`

Defined in: [src/clients/worker/worker.ts:49](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/worker/worker.ts#L49)

***

### registeredWorkflowPromises

> **registeredWorkflowPromises**: `Promise`\<`any`\>[] = `[]`

Defined in: [src/clients/worker/worker.ts:63](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/worker/worker.ts#L63)

***

### workerId

> **workerId**: `undefined` \| `string`

Defined in: [src/clients/worker/worker.ts:50](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/worker/worker.ts#L50)

***

### workflow\_registry

> **workflow\_registry**: ([`Workflow`](../interfaces/Workflow.md) \| [`WorkflowDefinition`](../type-aliases/WorkflowDefinition.md))[] = `[]`

Defined in: [src/clients/worker/worker.ts:55](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/worker/worker.ts#L55)

## Methods

### exitGracefully()

> **exitGracefully**(`handleKill`): `Promise`\<`void`\>

Defined in: [src/clients/worker/worker.ts:732](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/worker/worker.ts#L732)

#### Parameters

##### handleKill

`boolean`

#### Returns

`Promise`\<`void`\>

***

### getGroupKeyActionEvent()

> **getGroupKeyActionEvent**(`action`, `eventType`, `payload`): `GroupKeyActionEvent`

Defined in: [src/clients/worker/worker.ts:683](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/worker/worker.ts#L683)

#### Parameters

##### action

`Action`

##### eventType

`GroupKeyActionEventType`

##### payload

`any` = `''`

#### Returns

`GroupKeyActionEvent`

***

### getHandler()

> **getHandler**(`workflows`): `WebhookHandler`

Defined in: [src/clients/worker/worker.ts:121](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/worker/worker.ts#L121)

#### Parameters

##### workflows

[`Workflow`](../interfaces/Workflow.md)[]

#### Returns

`WebhookHandler`

***

### getStepActionEvent()

> **getStepActionEvent**(`action`, `eventType`, `shouldNotRetry`, `payload`, `retryCount`): `StepActionEvent`

Defined in: [src/clients/worker/worker.ts:661](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/worker/worker.ts#L661)

#### Parameters

##### action

`Action`

##### eventType

`StepActionEventType`

##### shouldNotRetry

`boolean`

##### payload

`any` = `''`

##### retryCount

`number` = `0`

#### Returns

`StepActionEvent`

***

### handleAction()

> **handleAction**(`action`): `Promise`\<`void`\>

Defined in: [src/clients/worker/worker.ts:796](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/worker/worker.ts#L796)

#### Parameters

##### action

`Action`

#### Returns

`Promise`\<`void`\>

***

### handleCancelStepRun()

> **handleCancelStepRun**(`action`): `Promise`\<`void`\>

Defined in: [src/clients/worker/worker.ts:702](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/worker/worker.ts#L702)

#### Parameters

##### action

`Action`

#### Returns

`Promise`\<`void`\>

***

### handleStartGroupKeyRun()

> **handleStartGroupKeyRun**(`action`): `Promise`\<`void`\>

Defined in: [src/clients/worker/worker.ts:571](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/worker/worker.ts#L571)

#### Parameters

##### action

`Action`

#### Returns

`Promise`\<`void`\>

***

### handleStartStepRun()

> **handleStartStepRun**(`action`): `Promise`\<`void`\>

Defined in: [src/clients/worker/worker.ts:439](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/worker/worker.ts#L439)

#### Parameters

##### action

`Action`

#### Returns

`Promise`\<`void`\>

***

### ~~register\_workflow()~~

> **register\_workflow**(`initWorkflow`): `Promise`\<`void`\>

Defined in: [src/clients/worker/worker.ts:142](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/worker/worker.ts#L142)

#### Parameters

##### initWorkflow

[`Workflow`](../interfaces/Workflow.md)

#### Returns

`Promise`\<`void`\>

#### Deprecated

use registerWorkflow instead

***

### registerAction()

> **registerAction**\<`T`, `K`\>(`actionId`, `action`): `void`

Defined in: [src/clients/worker/worker.ts:435](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/worker/worker.ts#L435)

#### Type Parameters

##### T

`T`

##### K

`K`

#### Parameters

##### actionId

`string`

##### action

[`StepRunFunction`](../type-aliases/StepRunFunction.md)\<`T`, `K`\>

#### Returns

`void`

***

### registerDurableActionsV1()

> **registerDurableActionsV1**(`workflow`): `void`

Defined in: [src/clients/worker/worker.ts:146](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/worker/worker.ts#L146)

#### Parameters

##### workflow

[`WorkflowDefinition`](../type-aliases/WorkflowDefinition.md)

#### Returns

`void`

***

### registerWebhook()

> **registerWebhook**(`webhook`): `Promise`\<`AxiosResponse`\<[`WebhookWorkerCreated`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WebhookWorkerCreated.md), `any`\>\>

Defined in: [src/clients/worker/worker.ts:135](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/worker/worker.ts#L135)

#### Parameters

##### webhook

[`WebhookWorkerCreateRequest`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WebhookWorkerCreateRequest.md)

#### Returns

`Promise`\<`AxiosResponse`\<[`WebhookWorkerCreated`](../Hatchet-TypeScript-SDK/namespaces/APIContracts/interfaces/WebhookWorkerCreated.md), `any`\>\>

***

### registerWorkflow()

> **registerWorkflow**(`initWorkflow`): `Promise`\<`void`\>

Defined in: [src/clients/worker/worker.ts:346](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/worker/worker.ts#L346)

#### Parameters

##### initWorkflow

[`Workflow`](../interfaces/Workflow.md)

#### Returns

`Promise`\<`void`\>

***

### registerWorkflowV1()

> **registerWorkflowV1**(`initWorkflow`): `Promise`\<`void`\>

Defined in: [src/clients/worker/worker.ts:186](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/worker/worker.ts#L186)

#### Parameters

##### initWorkflow

[`BaseWorkflowDeclaration`](BaseWorkflowDeclaration.md)\<`any`, `any`\>

#### Returns

`Promise`\<`void`\>

***

### start()

> **start**(): `Promise`\<`void`\>

Defined in: [src/clients/worker/worker.ts:756](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/worker/worker.ts#L756)

#### Returns

`Promise`\<`void`\>

***

### stop()

> **stop**(): `Promise`\<`void`\>

Defined in: [src/clients/worker/worker.ts:728](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/worker/worker.ts#L728)

#### Returns

`Promise`\<`void`\>

***

### upsertLabels()

> **upsertLabels**(`labels`): `Promise`\<`WorkerLabels`\>

Defined in: [src/clients/worker/worker.ts:811](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/worker/worker.ts#L811)

#### Parameters

##### labels

`WorkerLabels`

#### Returns

`Promise`\<`WorkerLabels`\>
