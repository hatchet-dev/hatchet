[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / SleepCondition

# Class: SleepCondition

Defined in: [src/v1/conditions/sleep-condition.ts:36](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/conditions/sleep-condition.ts#L36)

Represents a condition that waits for a specified duration before proceeding.
This condition is useful for implementing time delays in workflows.

## Example

```ts
// Create a condition that waits for 5 minutes
const waitCondition = new SleepCondition(
  "5m",
  "reminder_delay",
  () => console.log("Wait period completed!")
);
```

## Extends

- `Condition`

## Constructors

### Constructor

> **new SleepCondition**(`sleepFor`, `readableDataKey?`, `action?`): `SleepCondition`

Defined in: [src/v1/conditions/sleep-condition.ts:50](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/conditions/sleep-condition.ts#L50)

Creates a new sleep condition that waits for the specified duration.

#### Parameters

##### sleepFor

[`Duration`](../type-aliases/Duration.md)

Duration to wait in Go duration string format (e.g., "30s", "5m")

##### readableDataKey?

`string`

Optional unique identifier for the condition data.
                       When multiple sleep conditions have the same duration, using a custom
                       readableDataKey prevents duplicate data by differentiating between them.
                       If not provided, defaults to `sleep-${sleepFor}`.

##### action?

`Action`

Optional action to execute when the sleep duration completes

#### Returns

`SleepCondition`

#### Overrides

`Condition.constructor`

## Properties

### base

> **base**: `BaseCondition`

Defined in: [src/v1/conditions/base.ts:19](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/conditions/base.ts#L19)

#### Inherited from

`Condition.base`

***

### sleepFor

> **sleepFor**: [`Duration`](../type-aliases/Duration.md)

Defined in: [src/v1/conditions/sleep-condition.ts:38](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/conditions/sleep-condition.ts#L38)

The duration to sleep for in Go duration string format
