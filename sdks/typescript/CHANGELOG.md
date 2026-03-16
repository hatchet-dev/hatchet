# Changelog
All notable changes to Hatchet's TypeScript SDK will be documented in this changelog.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.17.0] - 2026-03-16

### Added

- OpenTelemetry instrumentation via `HatchetInstrumentor` with automatic tracing for workflow runs, event pushes, and step executions
- OpenTelemetry example demonstrating automatic and custom span instrumentation (`examples/opentelemetry_instrumentation`)

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
