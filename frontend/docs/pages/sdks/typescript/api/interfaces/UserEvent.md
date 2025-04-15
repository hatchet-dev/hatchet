[**Hatchet TypeScript SDK**](../README.md)

***

[Hatchet TypeScript SDK](../README.md) / UserEvent

# Interface: UserEvent

Defined in: [src/v1/conditions/user-event-condition.ts:3](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/conditions/user-event-condition.ts#L3)

## Properties

### eventKey

> **eventKey**: `string`

Defined in: [src/v1/conditions/user-event-condition.ts:9](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/conditions/user-event-condition.ts#L9)

The unique key identifying the specific user event to monitor.
This should match the event key that will be emitted from your application.

#### Example

```ts
"button:clicked", "page:viewed", "form:submitted"
```

***

### expression?

> `optional` **expression**: `string`

Defined in: [src/v1/conditions/user-event-condition.ts:16](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/conditions/user-event-condition.ts#L16)

Optional CEL expression to evaluate against the event data.
When provided, the condition will only trigger if this expression evaluates to true.

#### Example

```ts
"input.quantity > 5", "input.status == 'completed'"
```

***

### readableDataKey?

> `optional` **readableDataKey**: `string`

Defined in: [src/v1/conditions/user-event-condition.ts:25](https://github.com/hatchet-dev/hatchet/blob/0288a24f2e9f14787135b399bd47182f4d1260d9/sdks/typescript/src/v1/conditions/user-event-condition.ts#L25)

Optional unique identifier for the event data in the readable stream.
When multiple conditions are listening to the same event type,
using a custom readableDataKey prevents duplicate data processing
by differentiating between the conditions in the data store.
If not specified, the eventKey will be used as the default identifier.
