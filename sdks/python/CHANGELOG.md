# Changelog

All notable changes to Hatchet's Python SDK will be documented in this changelog.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.26.2] - 2026-02-26

### Added

- Adds `retry_transport_errors` and `retry_transport_methods` to `TenacityConfig` to optionally retry REST transport-level failures for configured HTTP methods (default: `GET`, `DELETE`). Default behavior is unchanged.

### Changed

- Uses a structured `http_method` on `RestTransportError` for determining retry eligibility.

## [1.26.1] - 2026-02-25

### Added

- Adds `retry_429` to `TenacityConfig` (default: `False`) to optionally retry REST HTTP 429 responses.
- Adds `TooManyRequestsException` and maps REST HTTP 429 responses to it.

## [1.26.0] - 2026-02-25

### Fixed

- Fixes dependencies not working when using `type Dependency = Annotated[..., ...]` syntax for annotations on python version 3.12 and 3.13. Adds `typing-inspection` as a dependency.

### Changed

- Changes one function in the python SDK to use `inspect.iscoroutinefunction` instead of `asyncio.iscoroutinefunction` which is deprecated.

## [1.25.2] - 2026-02-19

### Fixed

- Reverts cancellation changes in 1.25.0 that introduced a regression

## [1.25.1] - 2026-02-17

### Fixed

- Fixes internal registration of durable slots

## [1.25.0] - 2026-02-17 **YANKED ON 2/19/26**

### Added

- Adds a `CancellationToken` class for coordinating cancellation across async and sync operations. The token provides both `asyncio.Event` and `threading.Event` primitives, and supports registering child workflow run IDs and callbacks.
- Adds a `CancellationReason` enum with structured reasons for cancellation (`user_requested`, `timeout`, `parent_cancelled`, `workflow_cancelled`, `token_cancelled`).
- Adds a `CancelledError` exception (inherits from `BaseException`, mirroring `asyncio.CancelledError`) for sync code paths.
- Adds `cancellation_grace_period` and `cancellation_warning_threshold` configuration options to `ClientConfig` for controlling cancellation timing behavior.
- Adds `await_with_cancellation` and `race_against_token` utility functions for racing awaitables against cancellation tokens.
- The `Context` now exposes a `cancellation_token` property, allowing tasks to observe and react to cancellation signals directly.

### Changed

- The `Context.exit_flag` is now backed by a `CancellationToken` instead of a plain boolean. The property is maintained for backwards compatibility.
- Durable context `aio_wait_for` now respects the cancellation token, raising `asyncio.CancelledError` if the task is cancelled while waiting.

## [1.24.0] - 2026-02-13

### Added

- Webhooks client for managing incoming webhooks: create, list, get, update, and delete methods for webhooks, so external systems (e.g. GitHub, Stripe) can trigger workflows via HTTP.

## [1.23.4] - 2026-02-13

### Changed

- Fixes cases where raising exception classes or exceptions with no message would cause the whole error including stack trace to be converted to an empty string.
- When an error is raised because a workflow has no tasks it now includes the workflows name.

## [1.23.3] - 2026-02-12

### Added

- Adds type-hinted `Standalone.output_validator` and `Standalone.output_validator_type` properties to support easier type-safety and match the `input_validator` property pattern on `BaseWorkflow`.
- Adds type-hinted `Task.output_validator` and `Task.output_validator_type` properties to support easier type-safety and match the patterns on `BaseWorkflow/Standalone`.
- Adds parameterized unit tests documenting current retry behavior of the Python SDK’s tenacity retry predicate for REST and gRPC errors.

## [1.23.2] - 2026-02-11

### Changed

- Improves error handling for REST transport-level failures by raising typed exceptions for timeouts, connection, TLS, and protocol errors while preserving existing diagnostics.

## [1.23.1] - 2026-02-10

### Changed

- Fixes a bug introduced in v1.21.0 where the `BaseWorkflow.input_validator` class property became incorrectly typed. Now separate properties are available for the type adapter and the underlying type.

## [1.23.0] - 2026-02-05

### Internal Only

- Updated gRPC/REST contract field names to snake_case for consistency across SDKs.

## [1.22.16] - 2026-02-05

### Changed

- Changes the python SDK to use `inspect.iscoroutinefunction` instead of `asyncio.iscoroutinefunction` which is deprecated.
- Improves error diagnostics for transport-level failures in the REST client, such as SSL, connection, and timeout errors, by surfacing additional context.

## [1.22.15] - 2026-02-02

### Added

- Adds `task_name` and `workflow_name` properties to the `Context` and `DurableContext` classes to allow tasks and lifespans to access their own names.

### Changed

- Fixes a bug to allow `ContextVars` to be used in lifespans
- Improves worker shutdown + cleanup logic to avoid leaking semaphores in the action listener process.

## [1.22.14] - 2026-01-31

### Changed

- Allows `None` to be sent from `send_step_action_event` to help limit an internal error on the engine.

## [1.22.13] - 2026-01-29

### Added

- Sends the `task_retry_count` when sending logs to the engine to enable filtering on the frontend.

## [1.22.12] - 2026-01-28

### Added

- Adds a `default_additional_metadata` to the `hatchet.workflow`, `hatchet.task`, and `hatchet.durable_task` methods, which allows you to declaratively provide additional metadata that will be attached to each run of the workflow or task by default.

### Internal Only

- Sends a JSON schema to the engine on workflow registration in order to power autocomplete for triggering workflows from the dashboard.

## [1.22.11] - 2026-01-27

### Changed

- Improves handling of cancellations for tasks to limit how often tasks receive a cancellation but then are marked as succeeded anyways.

## [1.22.10] - 2026-01-26

### Added

- `HATCHET_CLIENT_WORKER_HEALTHCHECK_BIND_ADDRESS` now allows configuring the bind address for the worker healthcheck server (default: `0.0.0.0`)

## [1.22.9] - 2026-01-26

### Added

- Adds missing `unwrap` for `schedule_workflow` in OpenTelemetry instrumentor.

## [1.22.8] - 2026-01-20

### Added

- Adds `HATCHET_CLIENT_WORKER_HEALTHCHECK_EVENT_LOOP_BLOCK_THRESHOLD_SECONDS` to configure when the worker healthcheck becomes unhealthy if the listener process event loop is blocked / task runs are not starting promptly.

### Removed

- Removes a bunch of Poetry scripts that were mostly used for local development and are not necessary for end users of the SDK.

### Changed

- The worker healthcheck server (`/health`, `/metrics`) now runs in the spawned action-listener process (non-durable preferred; durable fallback), instead of the main worker process.
- The worker `/health` endpoint now checks for listener connection status and aio event loop health.
- The worker `/metrics` endpoint now exposes listener-focused metrics like `hatchet_worker_listener_health_<worker_name>` and `hatchet_worker_event_loop_lag_seconds_<worker_name>`.

## [1.22.7] - 2026-01-19

### Added

- Adds `is_in_hatchet_serialization_context` function which can be used on a Pydantic `ValidationInfo.context` to determine if the validation/serialization is occurring as a part of Hatchet deserializing task input or serializing task outputs.

## [1.22.6] - 2026-01-14

### Added

- Adds `max_attempts: int` (retries + 1) to the Context

## [1.22.5] - 2026-01-09

### Added

- Adds an `additional_metadata` field to the `get_details` response.

## [1.22.4] - 2026-01-08

### Added

- Adds a `get_details` method to the runs client

## [1.22.3] - 2026-01-07

### Changed

- Fixes an issue with the type signature for chained dependencies
- Truncates log messages to 10,000 characters to avoid issues with overly large logs.

## [1.22.2] - 2025-12-31

### Added

- Crons can now be provided by alias, e.g. `@daily`

### Changed

- Failed workflow logs are only reported at the `exception` level either on the last retry attempt or if the task is marked as `non_retryable`, to avoid spamming e.g. Sentry with exceptions.

## [1.22.1] - 2025-12-30

### Changed

- Regenerates some API signatures after deprecating many v0 routes.

## [1.22.0] - 2025-12-26

### Added

- Dependencies are now chainable, so one dependency can rely on an upstream one, similar to in FastAPI.
- Dependencies can now be both functions (sync and async) and context managers (sync and async) to allow for cleaning up things like database connections, etc.
- The `ClientConfig` has a new `Tenacity` object, which allows for specifying retry config.
- Concurrency limits can now be specified as integers, which will provide behavior equivalent to setting a constant key with a `GROUP_ROUND_ROBIN` strategy.

### Changed

- Improves the errors raised out of the sync `result` method on the `WorkflowRunRef` to be more in line with the async version, raising a `FailedTaskRunExceptionGroup` that contains all of the task run errors instead of just the first one.

### Internal

- Replaces manual validation logic with Pydantic's `TypeAdapter` for improved correctness and flexibility.

## [1.21.8] - 2025-12-26

### Changed

- Fixes a bug where static rate limits reset their own values to zero on task registration.

## [1.21.7] - 2025-12-15

### Added

- Adds a `get` method to the event client

## [1.21.6] - 2025-12-11

### Added

- Adds `get_task_stats` and `aio_get_task_stats` methods to the `metrics` feature client.

### Changed

- Regenerates the REST and gRPC clients to pick up latest API changes.

## [1.21.5] - 2025-12-06

### Changed

- Task outputs that fail to serialize to JSON will now raise an `IllegalTaskOutputError` instead of being stringified. This pulls errors from the engine upstream to the SDK, and will allow users to catch and handle these errors more easily.

## [1.21.4] - 2025-12-05

### Added

- Adds support for dynamic rate limits using CEL expressions (strings) for the `limit` parameter.

### Changed

- Fixes a serialization error caused by Pydantic sometimes being unable to encode bytes, reported here: https://github.com/hatchet-dev/hatchet/issues/2601
- Fixes a bug where string-based CEL expressions for `limit` were rejected due to the validation logic.

## [1.21.3] - 2025-11-26

### Added

- Adds GZIP compression for gRPC communication between the SDK and the Hatchet engine to reduce bandwidth usage.

## [1.21.2] - 2025-11-13

### Added

- Adds an OTel option to allow you to include the action name in the root span name for task runs.

### Changed

- Span kinds (e.g. producer, consumer) have been added to OpenTelemetry spans created by the SDK to better reflect their roles.

## [1.21.1] - 2025-11-08

### Changed

- The `list` methods for the logs client now allow for pagination via the `limit`, `since`, and `until` params.

## [1.21.0] - 2025-10-31

### Added

- Adds support for dataclasses as input validators for workflows (and tasks), and also as output validators for tasks.

### Changed

- Fixes a bug where an exception in a lifespan would cause the lifespan to hang indefinitely.

## [1.20.2] - 2025-10-15

### Added

- Adds a `include_payloads` parameter to the `list` methods on the runs client (defaults to true, so no change in behavior).

## [1.20.1] - 2025-10-14

### Added

- Adds wrapper methods for bulk cancelling / replaying large numbers of runs with pagination.

## [1.20.0] - 2025-10-3

### Removed

- Removes all references to `get_group_key_*` which is no longer available in V1
- Removes all checks + references to V0

## [1.19.0] - 2025-09-24

### Removed

- Removed the deprecated `v0` client and all related code.
- Removed unused dependencies.

## [1.18.1] - 2025-08-26

### Changed

- Fixes an install issue caused by a misnamed optional dependency.

## [1.18.0] - 2025-08-26

### Added

- Adds a `stubs` client on the main `Hatchet` client, which allows for creating typed stub tasks and workflows. These are intended to be used for triggering workflows that are registered on other workers in either other services or other languages.
- Adds a config option `force_shutdown_on_shutdown_signal` which allows users to forcefully terminate all processes when a shutdown signal is received instead of waiting for them to exit gracefully.

## [1.17.2] - 2025-08-20

### Added

- Adds back an optional `cel-python` dependency for v0 compatibility, allowing users to use the v0 client with the v0-compatible features in the SDK.
- Adds `dependencies` to the `mock_run` methods on the `Standalone`.
- Removes `aiostream` dependency that was unused.
- Removes `aiohttp-retry` dependency that was unused.

## [1.17.1] - 2025-08-18

### Added

- Adds a `HATCHET_CLIENT_LOG_QUEUE_SIZE` environment variable to configure the size of the log queue used for capturing logs and forwarding them to Hatchet

## [1.17.0] - 2025-08-12

### Added

- Adds support for dependency injection in tasks via the `Depends` class.
- Deprecated `fetch_task_run_error` in favor of `get_task_run_error`, which returns a `TaskRunError` object instead of a string. This allows for better error handling and debugging.

### Changed

- Uses `logger.exception` in place of `logger.error` in the action runner to improve (e.g.) Sentry error reporting
- Extends the `TaskRunError` to include the `task_run_external_id`, which is useful for debugging and tracing errors in task runs.
- Fixes an issue with logging which allows log levels to be respected over the API.

### Removed

- Removes the `cel-python` dependency

## [1.16.5] - 2025-08-07

### Changed

- Relaxes constraint on Prometheus dependency

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
