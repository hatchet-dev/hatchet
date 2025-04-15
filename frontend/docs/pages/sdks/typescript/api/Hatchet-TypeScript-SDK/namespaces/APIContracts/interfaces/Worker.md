[**Hatchet TypeScript SDK**](../../../../README.md)

***

[Hatchet TypeScript SDK](../../../../README.md) / [APIContracts](../README.md) / Worker

# Interface: Worker

Defined in: [src/clients/rest/generated/data-contracts.ts:1176](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1176)

## Properties

### actions?

> `optional` **actions**: `string`[]

Defined in: [src/clients/rest/generated/data-contracts.ts:1194](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1194)

The actions this worker can perform.

***

### availableRuns?

> `optional` **availableRuns**: `number`

Defined in: [src/clients/rest/generated/data-contracts.ts:1204](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1204)

The number of runs this worker can execute concurrently.

***

### dispatcherId?

> `optional` **dispatcherId**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1212](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1212)

the id of the assigned dispatcher, in UUID format

#### Format

uuid

#### Min Length

36

#### Max Length

36

#### Example

```ts
"bb214807-246e-43a5-a25d-41761d1cff9e"
```

***

### labels?

> `optional` **labels**: [`WorkerLabel`](WorkerLabel.md)[]

Defined in: [src/clients/rest/generated/data-contracts.ts:1214](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1214)

The current label state of the worker.

***

### lastHeartbeatAt?

> `optional` **lastHeartbeatAt**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1186](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1186)

The time this worker last sent a heartbeat.

#### Format

date-time

#### Example

```ts
"2022-12-13T15:06:48.888358-05:00"
```

***

### lastListenerEstablished?

> `optional` **lastListenerEstablished**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1192](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1192)

The time this worker last sent a heartbeat.

#### Format

date-time

#### Example

```ts
"2022-12-13T15:06:48.888358-05:00"
```

***

### maxRuns?

> `optional` **maxRuns**: `number`

Defined in: [src/clients/rest/generated/data-contracts.ts:1202](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1202)

The maximum number of runs this worker can execute concurrently.

***

### metadata

> **metadata**: [`APIResourceMeta`](APIResourceMeta.md)

Defined in: [src/clients/rest/generated/data-contracts.ts:1177](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1177)

***

### name

> **name**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1179](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1179)

The name of the worker.

***

### recentStepRuns?

> `optional` **recentStepRuns**: [`RecentStepRuns`](RecentStepRuns.md)[]

Defined in: [src/clients/rest/generated/data-contracts.ts:1198](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1198)

The recent step runs for the worker.

***

### runtimeInfo?

> `optional` **runtimeInfo**: [`WorkerRuntimeInfo`](WorkerRuntimeInfo.md)

Defined in: [src/clients/rest/generated/data-contracts.ts:1222](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1222)

***

### slots?

> `optional` **slots**: [`SemaphoreSlots`](SemaphoreSlots.md)[]

Defined in: [src/clients/rest/generated/data-contracts.ts:1196](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1196)

The semaphore slot state for the worker.

***

### status?

> `optional` **status**: `"ACTIVE"` \| `"INACTIVE"` \| `"PAUSED"`

Defined in: [src/clients/rest/generated/data-contracts.ts:1200](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1200)

The status of the worker.

***

### type

> **type**: `"SELFHOSTED"` \| `"MANAGED"` \| `"WEBHOOK"`

Defined in: [src/clients/rest/generated/data-contracts.ts:1180](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1180)

***

### webhookId?

> `optional` **webhookId**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1221](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1221)

The webhook ID for the worker.

#### Format

uuid

***

### webhookUrl?

> `optional` **webhookUrl**: `string`

Defined in: [src/clients/rest/generated/data-contracts.ts:1216](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L1216)

The webhook URL for the worker.
