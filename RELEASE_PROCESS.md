# Hatchet release process

Hatchet currently functions as a monorepo, where coupling can drastically vary across components.

The Python, Typescript, and Ruby SDKs are functionally isolated services, each with their own
changelog, versioning strategy, README, etc. -- making releasing new versions trivial.

In contrast, the boundary between Go components is far more blurred. Sharing a single `go.mod`
and multiple common packages (in `pkg` and `internal`) means that new releases of the API, engine,
and the Go SDK are required to be tagged, and released, under the same version. The Go SDK has no
separate publish step -- it is consumed directly via the platform's `v<x.y.z>` tag (`go get`
resolves against it), so it ships automatically with each platform release.

## Hatchet Platform

Hatchet platform releases come in two flavours, intermediate (1) and stable (2).

1. Correspond to a lightweight tag on a commit. Each intermediate release will be bundled
with new images pushed to our GitHub container registry at `ghcr.io/hatchet-dev/hatchet`.

2. Correspond to an annotated tag. Each stable release will bundle & push new packages to our
container registry at `ghcr.io/hatchet-dev/hatchet` (similar to `intermediate`), with the addition of a
new GitHub Release on our [releases page](https://github.com/hatchet-dev/hatchet/releases).
Release notes are generated via a combination of human-prose and automatically generated sections.

   The GitHub Release is created as a **draft** initially. Once reviewed and ready, it can be
   published manually, which sets it as `latest`.

### Generating Changelog entry

A pull request will only be included when generating release notes iff it is:

- Labeled with `engine` or `dashboard`.
- Scoped to include `engine`, `api`, `migrate`, `admin`, `cli`, `dashboard`, or `lite` (e.g. `fix(cli): ...`).

### Release Notes

The `CHANGELOG.md` contains user-facing changelog entries for each **stable** release. This should contain an overview of the changes introduced by
the stable release in addition to highlights, upgrade notes, breaking changes, or any other important information that we expect a user to care about
when upgrading.

Changes for an upcoming release can be drafted under the `[Unreleased]` section (atop `CHANGELOG.md`).

When ready to release, the unreleased section should be promoted to reflect the version and date in the form `[x.y.z] - YYYY-mm-dd`.

e.g.
```diff
- ## [Unreleased]
+ ## [0.90.0] - 2026-06-21
```

Then, when a new release is cut, the handwritten release notes are concatenated with the generated section, to create a single body for the GitHub release.

### Cutting a Release

1. Promote `[Unreleased]` in `CHANGELOG.md` to `[x.y.z] - YYYY-mm-dd` and merge to `main`.
2. Check out that commit and run `task release TAG=v<x.y.z>`. It tags `HEAD` with an annotated tag carrying
   the `CHANGELOG.md` entry, triggering the release process, and finally resulting in a the draft GitHub Release.

`task patch`/`task minor` are optional helpers that push a lightweight tag (an intermediate release) first;
bare `task release` then annotates whatever tag points at `HEAD`.

## Hatchet SDKs

Each SDK (Python, Typescript, Ruby) is released independently from its own directory, with its
own versioning strategy.

Similarly to the Hatchet Platform approach, changes can be drafted under `[Unreleased]` and
promoted to `[x.y.z] - YYYY-mm-dd` when ready to release. A GitHub release is then created
with release notes being a concatenation of the appropriate `CHANGELOG.md` entry and an automatically
derived from the commits/pull-request starting from the previous release.

A release means updating both the `CHANGELOG.md` and the manifest holding the version. On merge to
`main`, a CI job creates and pushes the SDK's tag to cut the release.

### Python

Version lives in `sdks/python/pyproject.toml`. Tagged as `py/<x.y.z>`.

```diff
  # sdks/python/CHANGELOG.md
- ## [Unreleased]
+ ## [0.0.1] - 2026-06-21
```

```diff
  # sdks/python/pyproject.toml
- version = "0.0.0"
+ version = "0.0.1"
```

CI then pushes `py/0.90.0`.

### Ruby

Version lives in `sdks/ruby/src/lib/hatchet/version.rb`. Tagged as `rb/<x.y.z>`.

```diff
  # sdks/ruby/src/CHANGELOG.md
- ## [Unreleased]
+ ## [0.0.1] - 2026-06-21
```

```diff
  # sdks/ruby/src/lib/hatchet/version.rb
- VERSION = "0.0.0"
+ VERSION = "0.0.1"
```

### Typescript

Version lives in `sdks/typescript/package.json`. Tagged as `ts/<x.y.z>`.

```diff
  # sdks/typescript/CHANGELOG.md
- ## [Unreleased]
+ ## [0.0.1] - 2026-06-21
```

```diff
  # sdks/typescript/package.json
- "version": "0.0.0",
+ "version": "0.0.1",
```
