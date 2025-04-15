[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / WorkflowDefinition

# Type Alias: WorkflowDefinition

> **WorkflowDefinition** = [`CreateWorkflowOpts`](CreateWorkflowOpts.md) & `object`

Defined in: [src/v1/declaration.ts:198](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/declaration.ts#L198)

Internal definition of a workflow and its tasks.

## Type declaration

### \_durableTasks

> **\_durableTasks**: [`CreateWorkflowDurableTaskOpts`](CreateWorkflowDurableTaskOpts.md)\<`any`, `any`\>[]

The durable tasks that make up this workflow.

### \_tasks

> **\_tasks**: [`CreateWorkflowTaskOpts`](CreateWorkflowTaskOpts.md)\<`any`, `any`\>[]

The tasks that make up this workflow.

### onFailure?

> `optional` **onFailure**: [`TaskFn`](TaskFn.md)\<`any`, `any`\> \| [`CreateOnFailureTaskOpts`](CreateOnFailureTaskOpts.md)\<`any`, `any`\>

(optional) onFailure handler for the workflow.
Invoked when any task in the workflow fails.

#### Param

The context of the workflow.

### onSuccess?

> `optional` **onSuccess**: [`TaskFn`](TaskFn.md)\<`any`, `any`\> \| [`CreateOnSuccessTaskOpts`](CreateOnSuccessTaskOpts.md)\<`any`, `any`\>

(optional) onSuccess handler for the workflow.
Invoked when all tasks in the workflow complete successfully.

#### Param

The context of the workflow.
