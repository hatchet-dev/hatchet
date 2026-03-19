## Hatchet SDK Changelog

All notable changes to Hatchet's Ruby SDK will be documented in this changelog.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2026-03-03

### Added

- Adds `desired_worker_labels` support to `trigger_workflow` and `bulk_trigger_workflow` to allow dynamically routing task runs to a specific worker at trigger time
- Cron expressions now support an optional leading seconds field (6-part expressions), e.g. `30 * * * * *` to trigger at 30 seconds past every minute.

## [0.1.1] - 2026-02-27

### Changed

- Updated internal dependencies to address security advisories.

## [0.1.0] - 2025-02-15

- Initial release of the Ruby SDK for Hatchet
- Task orchestration with simple tasks, DAGs, and child/fanout workflows
- Durable execution with durable tasks, durable events, and durable sleep
- Concurrency control (limit, round-robin, cancel in progress, cancel newest, multiple keys, workflow-level)
- Rate limiting
- Event-driven workflows
- Cron and scheduled workflows
- Retries with configurable backoff strategies
- Timeout management with refresh support
- On-failure and on-success callbacks
- Streaming support
- Webhook integration
- Bulk operations (fanout, replay)
- Priority scheduling
- Sticky and affinity worker assignment
- Deduplication
- Manual slot release
- Dependency injection
- Unit testing helpers
- Logging integration
- Run detail inspection
- RBS type signatures for IDE support
- REST and gRPC client support
