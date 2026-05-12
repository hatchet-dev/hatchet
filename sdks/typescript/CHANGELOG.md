# Changelog

All notable changes to Hatchet's TypeScript SDK will be documented in this changelog.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.22.1] - 2026-05-05

### Fixed

Moved optional dependencies from `optionalDependencies` to `peerDependencies`.

## [1.22.0] - 2026-04-28

### Added

- Adds `mcpTool` method to Workflow objects for integration with Claude and OpenAI agent SDKs. Requires Zod v4.
- Bumps minimum Zod version to `3.25.0`. Zod schemas provided to the SDK must be Zod 4 schemas, but you can still use Zod 3 in your application
code.

## [1.21.2] - 2026-04-22

### Added

- Adds `triggeringEventId` and `triggeringEventKey` to the `Context`

## [1.21.1] - 2026-04-21

### Fixed

- Adds an optional `label` on durable event waits, which will propagate through to the dashboard

## [1.21.0] - 2026-04-08

### Added

- runMany and runManyNoWait APIs for workflows and standalone tasks to support bulk runs with per-run options.
- RunManyOpt input shape containing an input object and an options object.

### Changed

- Bulk docs to include runMany and runManyNoWait examples.

## [1.20.1] - 2026-04-07

### Fixed

- Fixed duplicate child run deduplication when mixing `ctx.runChild()` and `workflow.runNoWait()` (or deeply nested recursive spawns). `Context.spawnIndex` and `ParentRunContextManager.childIndex` were tracked independently, causing both APIs to emit overlapping `childIndex` values and silently deduplicate children that should have been unique. The two counters now share a single source of truth via `AsyncLocalStorage`, and `incrementChildIndex` mutates the context object in place instead of replacing it with `enterWith`, which lost updates across `await` boundaries.

## [1.20.0] - 2026-04-07

### Added

- Adds `scope` and `lookbackWindow` arguments for the `DurableContext.waitForEvent`, which allows durable tasks to look back in time for events that may have been emitted before the task started.

## [1.19.1] - 2026-03-25

### Changed

- Event source info (`hatchet__source_workflow_run_id`, `hatchet__source_step_run_id`) is now injected into event metadata at the `EventClient` level, so cross-workflow trace linking works even without the OTel instrumentor enabled.

## [1.19.0] - 2026-03-25

### Fixed

- Fixed OpenTelemetry version mismatch causing `TypeError: Cannot read properties of undefined (reading 'name')` when exporting spans. The SDK now requires OpenTelemetry JS SDK 2.x (`@opentelemetry/sdk-trace-base@^2.0.0`, `@opentelemetry/core@^2.0.0`) to match the `@opentelemetry/exporter-trace-otlp-grpc@^0.208.0` dependency.

### Changed

- Updated OpenTelemetry optional dependencies to the unified 2.x release set.

## [1.18.0] - 2026-03-18

### Added

- OpenTelemetry instrumentation via `HatchetInstrumentor` with automatic tracing for workflow runs, event pushes, and step executions
- OpenTelemetry example demonstrating automatic and custom span instrumentation (`examples/opentelemetry_instrumentation`)

## [1.17.2] - 2026-03-17

### Added

- Added `getTaskStats` and `scrapePrometheusMetrics` methods to the metrics client.

## [1.17.1] - 2026-03-17

### Changed

- Updates the `DurableTaskRunAckEntryResult` interface to include `workflowRunExternalId` field, to enable spawning children from durable tasks fire-and-forget style.

## [1.17.0] - 2026-03-16

### Added

- Added a `DurableContext.waitForEvent` helper which returns the payload of the awaited event.
- Added an `EvictionPolicy`, which allows durable tasks to be evicted from the worker when idle.

### Changed

- Makes a bunch of internal-facing changes for new durable execution features

## [1.16.0] - 2026-03-11

### Added

- Added logs client for retrieving task run logs.

## [1.15.2] - 2026-03-06

### Fixed

- `waitFor` and task conditions (e.g. user event keys) are correctly namespaced when using a non-default namespace.
- Cron expressions now support an optional leading seconds field (6-part expressions), e.g. `30 * * * * *` to trigger at 30 seconds past every minute.

## [1.15.1] - 2026-03-04

### Fixed

- Fix npm publish so the package includes compiled JavaScript at the correct paths.

## [1.15.0] - 2026-03-03

### Added

- Adds a `desiredWorkerLabels` option to `RunOpts` to allow dynamically routing task runs to a specific worker at trigger time

## [1.14.0] - 2026-02-28

### Deprecated

- v0 SDK is now deprecated. Migrate to the v1 API for ongoing support.

### Added

- Internal legacy transformer for backwards compatibility with existing v0 workflows and workers.

## [1.13.1] - 2026-02-27

### Changed

- Updated internal dependencies to address security advisories.

## [1.13.0] - 2026-02-23

### Added

- Introduced client middleware support with composable `before`/`after` hooks to customize request handling and response processing.
- Added middleware examples and recipes to demonstrate practical client-side patterns.

## [1.12.1] - 2026-02-18

### Fixed

- Restored `ctx.taskRunId()` as a deprecated alias for `ctx.taskRunExternalId()` on both v0 and v1 worker contexts, so existing code calling `ctx.taskRunId()` continues to work after the proto naming changes in 1.11.0.

## [1.12.0] - 2026-02-13

### Added

- Webhooks client for managing incoming webhooks: create, list, get, update, and delete methods for webhooks, so external systems (e.g. GitHub, Stripe) can trigger workflows via HTTP.

## [1.11.0] - 2026-02-05

### Internal Only

- Updated gRPC/REST contract field names to lowerCamelCase for consistency across SDKs.

## [1.11.0] - 2026-02-04

### Changed

- Updated the metrics client for the latest server metrics APIs (including adding `getTaskStatusMetrics` for tenant task/run status counts).
- Removes deprecated metrics methods.

## [1.10.8] - 2026-02-02

### Changed

- Improved cancellation log messages: cancellation-related logs now use `debug` level instead of `error` level since cancellation is expected behavior, not a failure.
- Updated terminology in log messages from "step run" to "task run" for consistency.
- Added link to cancellation docs (https://docs.hatchet.run/home/cancellation) in error messages when task completion fails.

## [1.10.7] - 2026-01-27

### Added

- Adds support for an `inputValidator` prop on the various workflow definitions, e.g. `hatchet.workflow` and `hatchet.task`, which accepts a Zod schema to validate the input to the workflow or task. Used on the dashboard to provide autocomplete on the trigger workflow form.

## [1.10.6] - 2026-01-27

### Changed

- Improves handling of cancellations for tasks to limit how often tasks receive a cancellation but then are marked as succeeded anyways.
