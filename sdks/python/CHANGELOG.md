# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
