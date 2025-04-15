[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / Sleep

# Interface: Sleep

Defined in: [src/v1/conditions/sleep-condition.ts:4](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/conditions/sleep-condition.ts#L4)

## Properties

### readableDataKey?

> `optional` **readableDataKey**: `string`

Defined in: [src/v1/conditions/sleep-condition.ts:21](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/conditions/sleep-condition.ts#L21)

Optional unique identifier for the sleep condition in the readable stream.
When multiple conditions have the same sleep duration,
using a custom readableDataKey prevents duplicate data processing
by differentiating between the conditions in the data store.
If not specified, a default identifier based on the sleep duration will be used.

***

### sleepFor

> **sleepFor**: [`Duration`](../type-aliases/Duration.md)

Defined in: [src/v1/conditions/sleep-condition.ts:12](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/conditions/sleep-condition.ts#L12)

Amount of time to sleep for.
Specifies how long this condition should wait before proceeding.
Uses Go duration string format.

#### Example

```ts
"10s", "1m", "1m5s", "24h"
```
