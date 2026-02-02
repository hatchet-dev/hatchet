# Changelog

All notable changes to Hatchet's TypeScript SDK will be documented in this changelog.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.10.8] - 2026-02-02

### Changed

- Improved cancellation log messages: cancellation-related logs now use `debug` level instead of `error` level since cancellation is expected behavior, not a failure.
- Updated terminology in log messages from "step run" to "task run" for consistency.

## [1.10.7] - 2026-01-27

### Added

- Adds support for an `inputValidator` prop on the various workflow definitions, e.g. `hatchet.workflow` and `hatchet.task`, which accepts a Zod schema to validate the input to the workflow or task. Used on the dashboard to provide autocomplete on the trigger workflow form.

## [1.10.6] - 2026-01-27

### Changed

- Improves handling of cancellations for tasks to limit how often tasks receive a cancellation but then are marked as succeeded anyways.
