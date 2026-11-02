# Changelog

All notable changes to `@hatchet-dev/sandbox`, the TypeScript SDK.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Each release reads its notes from the matching section here.

## [Unreleased]

## [0.1.2] - 2026-11-02

### Added

- `create({ provider })` picks the sandbox provider by name when a tenant has more than one. It is sent as the `provider` metadata tag, so providers that already read that tag need no change.

### Changed

- The package now lives in [hatchet-dev/hatchet](https://github.com/hatchet-dev/hatchet) under `sdks/typescript-sandbox` and ships CommonJS alongside ESM. The wire contract with providers is unchanged.

### Fixed

- `files.write` of a staged file replayed in a durable task after its signed upload URL expired now asks the provider for a new URL instead of failing with `staging_failed`, matching what `files.read` already did.

## [0.1.1] - 2026-10-29

### Added

- `create({ from, inheritParentMetadata: true })` copies the parent sandbox's metadata onto the fork; tags given in `metadata` take precedence.
- `hatchet.sandbox.delete({ sandboxId?, metadata? })` and `sandbox.delete()`: remove sandboxes together with every snapshot and file stored for them. `kill()` keeps snapshots as before.

### Fixed

- `files.read` of a staged file replayed in a durable task after its signed URL expired now asks the provider to sign the same file again instead of failing with `staging_failed`.

## [0.1.0] - 2026-10-27

### Added

- `hatchet.sandbox.create`, `connect`, and `list` on the Hatchet client, by importing `@hatchet-dev/sandbox`.
- `Sandbox.exec` with `timeoutMs`, returning stdout, stderr, exit code, and per-step timings; `Sandbox.kill`.
- `Sandbox.files`: read, write, writeFiles, list, exists, getInfo, makeDir, rename, copy, remove, download, upload. Files over 2 MiB are staged through the provider's object store on presigned URLs.
- `create({ id, template, from, metadata, idleTimeoutMs })` for caller-chosen ids, images, forks, and expiry.
- Every call is a Hatchet task run, and a durable step inside a durable task.
- OpenTelemetry instrumentation, off with `BRICKIE_OTEL=false`.
