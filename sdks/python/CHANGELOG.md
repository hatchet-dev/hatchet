# Changelog

All notable changes to Hatchet's Python SDK will be documented in this changelog.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.13.0] - 2025-06-25

### Added

- Documentation for the `Context` classes
- Allows for a worker to be terminated after a certain number of tasks by providing the `terminate_worker_after_num_tasks` config option

### Changed

- Adds a number of helpful Ruff linting rules
- `DedupeViolationErr` is now `DedupeViolationError`
- Fixed events documentation to correctly have a skipped run example.
- Changed default arguments to many methods from mutable defaults like `[]` to None
- Changes `JSONSerializableMapping` from `Mapping` to `dict`
- Handles some potential bugs related to `asyncio` tasks being garbage collected.
- Improves exception printing with an `ExceptionGroup` implementation
- Fixes a bug with namespacing of user event conditions where the namespace was not respected so the task waiting for it would hang
- Fixes a memory leak in streaming and logging, and fixes some issues with log capture.

## [1.12.2] - 2025-06-17

### Changed

- Fixes a security vulnerability by bumping the `protobuf` library

## [1.12.1] - 2025-06-13

### Added

- Adds corresponding SDK changes from API changes to events (additional parameters to filter events by, additional data returned)

## [1.12.0] - 2025-06-06

### Added

- Adds a warning on client init if the SDK version is not compatible with the tenant (engine) version.
- Adds a `default_filters` parameter to the `Hatchet.workflow` and `Hatchet.task` methods to allow you to declaratively provide a list of filters that will be applied to the workflow by default when events are pushed.
- Adds `get_status` and `aio_get_status` methods to the `Runs` feature client, which return a workflow run's status by its ID.
- Adds a `update` methods to the `Filters` feature client.

### Changed

- Allows the `concurrency` parameter to tasks to be a `list`.
- Fixes an internal bug with duplicate concurrency expressions being set when using `Hatchet.task`.
- Modifies existing `datetime` handling to use UTC timestamps everywhere.

## [1.11.1] - 2025-06-05

### Changed

- Fixes a couple of blocking calls buried in the admin client causing loop blockages on child spawning

## [1.11.0] - 2025-05-29

### Changed

- Significant improvements to the OpenTelemetry instrumentor, including:
  - Traceparents are automatically propagated through the metadata now so the client does not need to provide them manually.
  - Added a handful of attributes to the `run_workflow`, `push_event`, etc. spans, such as the workflow being run / event being pushed, the metadata, and so on. Ignoring
  - Added tracing for workflow scheduling

## [1.10.2] - 2025-05-19

### Changed

- Fixing an issue with the spawn index being set at the `workflow_run_id` level and not the `(workflow_run_id, retry_count)` level, causing children to be spawned multiple times on retry.

## [1.10.1] - 2025-05-16

### Added

- Adds an `otel` item to the `ClientConfig` and a `excluded_attributes: list[OTelAttribute]` there to allow users to exclude certain attributes from being sent to the OpenTelemetry collector.

## [1.10.0] - 2025-05-16

### Added

- The main `Hatchet` client now has a `filters` attribute (a `Filters` client) which wraps basic CRUD operations for managing filters.
- Events can now be pushed with a `priority` attribute, which sets the priority of the runs triggered by the event.
- There are new `list` and `aio_list` methods for the `Events` client, which allow listing events.
- Workflow runs can now be filtered by `triggering_event_external_id`, to allow for seeing runs triggered by a specific event.
- There is now an `id` property on all `Workflow` objects (`Workflow` created by `hatchet.workflow` and `Standalone` created by `hatchet.task`) that returns the ID (UUID) of the workflow.
- Events can now be pushed with a `scope` parameter, which is required for using filters to narrow down the filters to consider applying when triggering workflows from the event.

### Changed

- The `name` parameter to `hatchet.task` and `hatchet.durable_task` is now optional. If not provided, the task name will be the same as the function name.
