---
applyTo: "**/*.go"
---

# Analytics Instrumentation

All new features MUST be instrumented with analytics. Every user-facing action and
every system-level operation that creates, mutates, or deletes a resource must emit
an analytics event. Choose the correct method based on throughput.

## Two Methods

### `Enqueue()` — Low-Throughput Features

Use for actions triggered directly by a human (UI clicks, API calls, CLI commands)
that occur at most a few times per minute per tenant. Each call sends one PostHog
event immediately.

**Examples of when to use:** tenant creation, user login/signup, workflow CRUD, API token
management, invite accept/reject, manual cancellation or
replay of individual runs.

**Signature:**

```go
analytics.Enqueue(ctx, resource, action, resourceID, properties)
```

User, tenant, and token identity are resolved automatically from `ctx`. Auth
middleware binds them via typed context keys (`analytics.TenantIDKey`,
`analytics.UserIDKey`, `analytics.APITokenIDKey`). Callers never pass identity
explicitly.

**Rules:**

- Always pass a meaningful `resourceID` (the UUID of the created/affected resource).
- Properties map: include fields that help segment usage (e.g. `"name"`, `"slug"`,
  `"source"`). Do NOT include PII beyond what is already in the user profile.
  Prefer lossy boolean flags (`has_priority`) over raw values when possible.

**Example (from tenant creation):**

```go
t.config.Analytics.Enqueue(
    ctx.Request().Context(),
    analytics.Tenant, analytics.Create,
    tenantID.String(),
    map[string]interface{}{
        "name": tenant.Name,
        "slug": tenant.Slug,
    },
)
```

### `Count()` — High-Throughput Features

Use for operations that can fire hundreds or thousands of times per second across
tenants: event ingestion, log ingestion, stream events, worker heartbeats, step run
state transitions, workflow run creation from schedules/crons, bulk replays.

`Count()` is non-blocking. Events are aggregated in memory by
`(resource, action, tenant, token, properties hash)` and flushed to PostHog every
30 seconds as a single event with a `count` field.

**Signature:**

```go
analytics.Count(ctx, resource, action, props...)
```

Tenant identity is resolved from `ctx` automatically (same as `Enqueue`).

**Rules:**

- Properties should capture boolean feature flags and categorical dimensions that
  are useful for segmentation, NOT unique identifiers. Each distinct set of property
  values creates a separate aggregation bucket; keep cardinality low.
- Use `analytics.Props(k1, v1, k2, v2, ...)` to build the property map. Nil values
  are automatically omitted.
- Name boolean properties with a `has_` prefix (e.g. `has_priority`, `has_labels`,
  `has_concurrency`).

**Example (from event ingestion):**

```go
i.analytics.Count(ctx, analytics.Event, analytics.Create, analytics.Props(
    "has_priority",        req.Priority != nil,
    "has_scope",           req.Scope != nil,
    "has_additional_meta", req.AdditionalMetadata != nil,
))
```

## Adding a New Resource or Action

If the existing constants in `pkg/analytics/analytics.go` do not cover the new
feature, add new `Resource` or `Action` constants there. Follow the existing
naming conventions:

- **Resources** are lowercase, hyphen-separated nouns: `task-run`, `workflow-run`,
  `webhook-worker`, `stream-event`.
- **Actions** are lowercase single verbs: `create`, `delete`, `cancel`, `replay`,
  `register`, `subscribe`, `listen`, `release`, `refresh`, `send`.

The resulting PostHog event name is `{resource}:{action}` (e.g. `task-run:cancel`).

## Where Analytics Lives

- Interface and constants: `pkg/analytics/analytics.go`
- Aggregator (batching for `Count`): `pkg/analytics/aggregating.go`
- PostHog implementation: `pkg/analytics/posthog/posthog.go`
- `NoOpAnalytics` (used when PostHog is disabled): `pkg/analytics/analytics.go`

Services receive an `Analytics` instance via their config struct. Access it as
`s.config.Analytics` (or the equivalent field name in the service).

## Checklist for New Features

1. Determine throughput: is the action user-initiated (low) or system/SDK-initiated
   at scale (high)?
2. Pick `Enqueue` or `Count` accordingly.
3. Add `Resource`/`Action` constants if needed.
4. Include meaningful properties — boolean `has_*` flags for feature detection,
   categorical fields for segmentation. Keep property cardinality low for `Count`.
5. Place the analytics call as close to the success path as possible (after
   validation, after the operation succeeds, not on error paths).
6. Verify the event appears in PostHog in a dev environment before merging.
