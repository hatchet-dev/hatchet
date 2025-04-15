[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / UserEventCondition

# Class: UserEventCondition

Defined in: [src/v1/conditions/user-event-condition.ts:50](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/conditions/user-event-condition.ts#L50)

Represents a condition that is triggered based on a specific user event.
This condition monitors for events with the specified key and evaluates
any provided expression against the event data.

## Param

The key identifying the specific user event to monitor

## Param

The CEL expression to evaluate against the event data

## Param

Optional parameter that provides a unique identifier for the data.
                       When multiple conditions are listening to the same event, using a custom
                       readableDataKey prevents duplicate data by differentiating between the conditions.
                       If not provided, defaults to the eventKey.

## Param

Optional action to execute when the condition is met

## Example

```ts
// Create a condition that triggers when a "purchase" event occurs with amount > 100
const purchaseCondition = new UserEventCondition(
  "purchase",
  "data.amount > 100",
  "high_value_purchase",
  () => console.log("High value purchase detected!")
);
```

## Extends

- `Condition`

## Constructors

### Constructor

> **new UserEventCondition**(`eventKey`, `expression`, `readableDataKey?`, `action?`): `UserEventCondition`

Defined in: [src/v1/conditions/user-event-condition.ts:54](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/conditions/user-event-condition.ts#L54)

#### Parameters

##### eventKey

`string`

##### expression

`string`

##### readableDataKey?

`string`

##### action?

`Action`

#### Returns

`UserEventCondition`

#### Overrides

`Condition.constructor`

## Properties

### base

> **base**: `BaseCondition`

Defined in: [src/v1/conditions/base.ts:19](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/conditions/base.ts#L19)

#### Inherited from

`Condition.base`

***

### eventKey

> **eventKey**: `string`

Defined in: [src/v1/conditions/user-event-condition.ts:51](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/conditions/user-event-condition.ts#L51)

***

### expression

> **expression**: `string`

Defined in: [src/v1/conditions/user-event-condition.ts:52](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/conditions/user-event-condition.ts#L52)
