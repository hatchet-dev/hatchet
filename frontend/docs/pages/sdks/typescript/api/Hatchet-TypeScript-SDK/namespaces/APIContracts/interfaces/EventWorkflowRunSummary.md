[**Hatchet TypeScript SDK**](../../../../README.md)

***

[Hatchet TypeScript SDK](../../../../README.md) / [APIContracts](../README.md) / EventWorkflowRunSummary

# Interface: EventWorkflowRunSummary

Defined in: [src/clients/rest/generated/data-contracts.ts:630](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L630)

## Properties

### cancelled?

> `optional` **cancelled**: `number`

Defined in: [src/clients/rest/generated/data-contracts.ts:660](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L660)

The number of cancelled runs.

#### Format

int64

***

### failed?

> `optional` **failed**: `number`

Defined in: [src/clients/rest/generated/data-contracts.ts:655](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L655)

The number of failed runs.

#### Format

int64

***

### pending?

> `optional` **pending**: `number`

Defined in: [src/clients/rest/generated/data-contracts.ts:635](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L635)

The number of pending runs.

#### Format

int64

***

### queued?

> `optional` **queued**: `number`

Defined in: [src/clients/rest/generated/data-contracts.ts:645](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L645)

The number of queued runs.

#### Format

int64

***

### running?

> `optional` **running**: `number`

Defined in: [src/clients/rest/generated/data-contracts.ts:640](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L640)

The number of running runs.

#### Format

int64

***

### succeeded?

> `optional` **succeeded**: `number`

Defined in: [src/clients/rest/generated/data-contracts.ts:650](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/clients/rest/generated/data-contracts.ts#L650)

The number of succeeded runs.

#### Format

int64
