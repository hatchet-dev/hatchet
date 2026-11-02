# Changelog

All notable changes to `@hatchet-dev/serverless` will be documented in this changelog.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2026-10-28

### Added

- `clientFromContext(ctx).streams.subscribe(topic, options)` reads a durable stream topic over the invocation socket. `streams.events()` is now an alias of it and will be removed in a future release.

## [0.1.0] - 2026-10-23

### Added

- Serverless task handlers for Cloudflare Workers, Vercel and Node, driven by the in-engine serverless operator: signed POST delivery for pure tasks (`hatchetClient: false`) and the invocation websocket for every other task, including durable tasks with sleeps, waits and child runs.
- `clientFromContext(ctx)`: child runs, durable stream topics and run events over the socket, with `refreshTimeout` and `releaseSlot` through the SDK's `/core` client.
- `@hatchet-dev/serverless/testing`: an in-process operator for tests without Hatchet.
