[**Hatchet TypeScript SDK**](../../../../README.md)

***

[Hatchet TypeScript SDK](../../../../README.md) / [APIContracts](../README.md) / JobRun

# Interface: JobRun

Defined in: [src/clients/rest/generated/data-contracts.ts:1007](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1007)

## Properties

### cancelledAt?

> `optional` **cancelledAt**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1025](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1025)

#### Format

date-time

***

### cancelledError?

> `optional` **cancelledError**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1027](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1027)

***

### cancelledReason?

> `optional` **cancelledReason**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1026](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1026)

***

### finishedAt?

> `optional` **finishedAt**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1021](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1021)

#### Format

date-time

***

### job?

> `optional` **job**: [`Job`](Job.md)

Defined in: [src/clients/rest/generated/data-contracts.ts:1013](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1013)

***

### jobId

> **jobId**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1012](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1012)

***

### metadata

> **metadata**: [`APIResourceMeta`](APIResourceMeta.md)

Defined in: [src/clients/rest/generated/data-contracts.ts:1008](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1008)

***

### result?

> `optional` **result**: `object`

Defined in: [src/clients/rest/generated/data-contracts.ts:1017](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1017)

***

### startedAt?

> `optional` **startedAt**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1019](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1019)

#### Format

date-time

***

### status

> **status**: [`JobRunStatus`](../enumerations/JobRunStatus.md)

Defined in: [src/clients/rest/generated/data-contracts.ts:1016](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1016)

***

### stepRuns?

> `optional` **stepRuns**: [`StepRun`](StepRun.md)[]

Defined in: [src/clients/rest/generated/data-contracts.ts:1015](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1015)

***

### tenantId

> **tenantId**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1009](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1009)

***

### tickerId?

> `optional` **tickerId**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1014](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1014)

***

### timeoutAt?

> `optional` **timeoutAt**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1023](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1023)

#### Format

date-time

***

### workflowRun?

> `optional` **workflowRun**: [`WorkflowRun`](WorkflowRun.md)

Defined in: [src/clients/rest/generated/data-contracts.ts:1011](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1011)

***

### workflowRunId

> **workflowRunId**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1010](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1010)
