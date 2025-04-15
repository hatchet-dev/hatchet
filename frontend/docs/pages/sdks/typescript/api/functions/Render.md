[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / Render

# Function: Render()

> **Render**(`action?`, `conditions?`): `Condition`[]

Defined in: [src/v1/conditions/index.ts:37](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/conditions/index.ts#L37)

Creates a condition that waits for all provided conditions to be met (AND logic)
use Or() to create a condition that waits for any of the provided conditions to be met (OR logic)

## Parameters

### action?

`Action`

### conditions?

Conditions or OrConditions to be rendered

[`Conditions`](../type-aliases/Conditions.md) | [`Conditions`](../type-aliases/Conditions.md)[]

## Returns

`Condition`[]

A flattened array of Conditions

## Example

```ts
const conditions = Render(
  Or({ sleepFor: 5 }, { eventKey: 'user:update' }),
  { eventKey: 'user:create' },
  Or({ eventKey: 'user:update' }, { eventKey: 'user:delete' })
);
```
