# Changelog

All notable changes to `@hatchet-dev/sandbox`, the TypeScript SDK.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Each release reads its notes from the matching section here.

## [Unreleased]

## [0.1.3] - 2026-11-09

### Added

- `create({ resources })` asks the provider for a machine size, `{ vcpu, memory, disk }`. `memory` and `disk` take a number of MiB or a Kubernetes quantity such as `'512Mi'`, `'4Gi'` or `'8G'`, and travel as `memoryMib` and `diskMib` on the wire. A provider that cannot honour a field fails the create with its own message, so plan limits show up as the task error. A provider that does not know the option ignores it. On Hatchet Cloud, Archil sizes the sandbox on create and forks keep the source's size, so `resources` with `from` is rejected there. Daytona and Freestyle resize after create, which on Daytona needs resizing enabled for the organization and on Freestyle can only grow. E2B sizes on the template and rejects the option.

## [0.1.2] - 2026-11-03

### Added

- `create({ provider })` picks the sandbox provider by name when a tenant has more than one. It is sent as the `provider` metadata tag, so providers that already read that tag need no change.
- The SDK works against any sandbox provider. On first use it checks whether the tenant serves the `hatchet-sandbox` task and uses that; otherwise it keeps using `brickie-sandbox`. Existing deployments need no change. A tenant should serve one of the two: turning on `hatchet-sandbox` while durable tasks that already called `brickie-sandbox` are in flight makes those tasks fail on resume with a non-determinism error, so drain them first.
- `HATCHET_SANDBOX_OTEL=false` turns the OpenTelemetry instrumentor off. `BRICKIE_OTEL=false` still works.

### Changed

- The package declares `sideEffects: true` so bundlers keep the import that adds `hatchet.sandbox` to the client.
- Run metadata tags are now `hatchet.sandbox.fn`, `hatchet.sandbox.id`, `hatchet.sandbox.image` and `hatchet.sandbox.from` (were `brickie.fn`, `brickie.sandbox`, `brickie.image`, `brickie.from`). Update any dashboard or run filter that keys on the old names.

### Fixed

- `files.write` of a staged file replayed in a durable task after its signed upload URL expired now asks the provider for a new URL instead of failing with `staging_failed`. For both `write` and `read`, that renewal is a plain task run rather than a durable child, so it does not shift the replayed child sequence.
- `files.write` with several files over 2 MiB converts every input first and then takes staged keys in file order, so a durable replay assigns the same key to the same file.

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
