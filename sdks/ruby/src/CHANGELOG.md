## Hatchet SDK Changelog

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
