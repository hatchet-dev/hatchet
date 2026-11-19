# Changelog

All notable changes to `@hatchet-dev/pi-durable`.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Each release reads its notes from the matching section here.

## [Unreleased]

## [0.1.0] - 2026-11-16

### Added

- `defineSessionTask` hosts one pi-durable session per Hatchet task run, with concurrency on the session id so a session has one owner at a time, retries that move the session to a new worker after a crash, and a scheduled run for the session's next wake-up.
- `SessionClient` creates and wakes sessions, sends `input`, `fork` and `abort` messages to a session's inbox stream, and reads its `SessionView` stream.
- `sandboxCoding()` installs pi's read, write, edit and bash tools, and `sandboxEnvs()` runs each conversation's tool calls in its own Hatchet sandbox. A fork starts from the sandbox snapshot taken at the fork point.
- `SandboxEnv`, a pi-durable `ExecutionEnv` backed by a Hatchet sandbox. `PI_SANDBOX_TESTS=1 pnpm test` runs pi-durable's `ExecutionEnv` conformance suite against it on a real sandbox.
- Needs `@hatchet-dev/sandbox` 0.1.5 or newer: commands run through `Sandbox.exec`, on whichever sandbox task the tenant serves. Aborting a command stops waiting for it, but the command runs on in the sandbox until it ends or times out.
