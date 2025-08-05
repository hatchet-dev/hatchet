# Changelog

All notable changes to Hatchet's Python SDK will be documented in this changelog.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.17.0] - 2025-07-31

### Added

- Adds support for dependency injection in tasks via the `Depends` class.

### Changed

- Uses `logger.exception` in place of `logger.error` in the action runner to improve (e.g.) Sentry error reporting
- Improves shutdown handlers to avoid deadlocks under load

## [1.16.4] - 2025-07-28

### Added

- Adds a new config option `grpc_enable_fork_support` to allow users to enable or disable gRPC fork support. This is useful for environments where gRPC fork support is not needed or causes issues. Previously was set to `False` by default, which would cause issues with e.g. Gunicorn setups. Can also be set with the `HATCHET_CLIENT_GRPC_ENABLE_FORK_SUPPORT` environment variable.

### Changed

- Changes `ValidTaskReturnType` to allow `Mapping[str, Any]` instead of `dict[str, Any]` to allow for more flexible return types in tasks, including using `TypedDict`.

## [1.16.3] - 2025-07-23

### Added

- Adds support for filters and formatters in the logger that's passed to the Hatchet client.
- Adds a flag to disable log capture.

### Changed

- Fixes a bug in `aio_sleep_for` and the `SleepCondition` that did not allow duplicate sleeps to be awaited correctly.
- Stops retrying gRPC requests on 4XX failures, since retrying won't help

## [1.16.2] - 2025-07-22

### Added

- Adds an `input_validator` property to `BaseWorkflow` which returns a typechecker-aware version of the validator class.

## [1.16.1] - 2025-07-18

### Added

- Adds a `CEL` feature client for debugging CEL expressions

## [1.16.0] - 2025-07-17

### Added

- Adds new methods for unit testing tasks and standalones, called `mock_run` and `aio_mock_run`, which allow you to run tasks and standalones in a mocked environment without needing to start a worker or connect to the engine.
- Improves exception logs throughout the SDK to provide more context for what went wrong when an exception is thrown.
- Adds `create_run_ref`, `get_result`, and `aio_get_result` methods to the `Standalone` class, to allow for getting typed results of a run more easily.
- Adds `return_exceptions` option to the `run_many` and `aio_run_many` methods to be more similar to e.g. `asyncio.gather`. If `True`, exceptions will be returned as part of the results instead of raising them.

### Changed

- Correctly propagates additional metadata through the various `run` methods to spawned children.

## [1.15.3] - 2025-07-14

### Changed

- `remove_null_unicode_character` now accepts any type of data, not just strings, dictionaries, lists, and tuples. If the data is not one of these types, it's returned as-is.

## [1.15.2] - 2025-07-12

### Changed

- Fixes an issue in `capture_logs` where the `log` call was blocking the event loop.

## [1.15.1] - 2025-07-11

### Added

- Correctly sends SDK info to the engine when a worker is created

## [1.15.0] - 2025-07-10

### Added

- The `Metrics` client now includes a method to scrape Prometheus metrics from the tenant.

### Changed

- The `Metrics` client's `get_task_metrics` and `get_queue_metrics` now return better-shaped, correctly-fetched data from the API.

## [1.14.4] - 2025-07-09

### Added

- Adds `delete` and `aio_delete` methods to the workflows feature client and the corresponding `Workflow` and `Standalone` classes, allowing for deleting workflows and standalone tasks.

## [1.14.3] - 2025-07-07

### Added

- Adds `remove_null_unicode_character` utility function to remove null unicode characters from data structures.

### Changed

- Task outputs that contain a null unicode character (\u0000) will now throw an exception instead of being serialized.
- OpenTelemetry instrumentor now correctly reports exceptions raised in tasks to the OTel collector.

## [1.14.2] - 2025-07-03

### Added

- The `Runs` client now has `list_with_pagination` and `aio_list_with_pagination` methods that allow for listing workflow runs with internal pagination. The wrappers on the `Standalone` and `Workflow` classes have been updated to use these methods.
- Added retries with backoff to all of the REST API wrapper methods on the feature clients.

## [1.14.1] - 2025-07-03

### Changed

- `DurableContext.aio_wait_for` can now accept an or group, in addition to sleep and event conditions.

## [1.14.0] - 2025-06-25

### Added

- Adds an `IllegalTaskOutputError` that handles cases where tasks return invalid outputs.
- Logs `NonRetryableException` as an info-level log so it doesn't get picked up by Sentry and similar tools.

### Changed

- Exports `NonRetryableException` at the top level
- Fixes an issue with the `status` field throwing a Pydantic error when calling `worker.get`
- Fixes an issue with duplicate protobufs if you try to import both the v1 and v0 clients.

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

## [1.12.3] - 2025-06-25

### Changed

- Fixes a namespacing-related but in the `workflow.id` property that incorrectly (and inconsistently) returned incorrect IDs for namespaced workflows.

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
