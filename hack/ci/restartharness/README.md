# Restart harness

Measures how much a `hatchet-engine` restart disrupts tenants, as one number
that a person or an agent can drive down and that a pull request can be judged
by.

`run` starts one engine build against Postgres and RabbitMQ, drives
multi-tenant load at it, restarts it with `SIGTERM` several times and writes a
results file. `compare` scores two results files against each other.

## Run it locally

```sh
docker compose up -d postgres rabbitmq

go build -o /tmp/rh/harness ./hack/ci/restartharness
go build -o /tmp/rh/head/ ./cmd/hatchet-engine ./cmd/hatchet-migrate ./cmd/hatchet-admin

/tmp/rh/harness run \
  --engine-bin /tmp/rh/head/hatchet-engine \
  --migrate-bin /tmp/rh/head/hatchet-migrate \
  --admin-bin /tmp/rh/head/hatchet-admin \
  --track mt-skewed --out /tmp/rh/head.json

/tmp/rh/harness compare /tmp/rh/base.json /tmp/rh/head.json
```

`run` takes binaries, not a checkout, so the base can be built once from
another commit (`git worktree add`, then the same `go build`) and its results
file kept while only the head is rebuilt and rerun. The schema is migrated and
the tenants, keys and tokens are created with the build's own `hatchet-migrate`
and `hatchet-admin`, so the two sides may differ in migrations or dependencies.

Without `--track`, `run` runs every track in turn into one file. A track takes
about five minutes. Each track gets a fresh database and a fresh RabbitMQ
vhost, both removed afterwards. The engine's and the workers' logs are written
next to `--out`.

The last line `compare` prints is `SCORE <seconds>`, or `SCORE INVALID`. Lower
is better and 0 is perfect. To hill-climb, keep one base results file, change
the engine, rebuild, rerun the track you care about and read that line.

## What is measured

Every tenant is offered probes at fixed timestamps, whatever happened to the
earlier ones: no retries, a fixed deadline. Each probe is two operations:

- an API call: the trigger RPC, bad if it fails or takes longer than the
  track's `APIDeadlineS`;
- a trigger→assigned: the task starting on the tenant's worker, bad if that
  takes longer than `AssignDeadlineS` from the probe's timestamp.

For each restart, over the `WindowS` seconds after `SIGTERM`:

```
lost seconds = WindowS × mean over tenants of (bad operations / offered operations)
```

This is the time an average tenant was down. Every tenant counts the same, so
a large tenant cannot hide small ones being starved. A track's number is the
mean over its restarts, and `SCORE` is the sum over tracks.

A run is `INVALID`, whatever its number, if:

- an accepted trigger never ran, or any trigger ran more than once;
- a task started while `max_runs` tasks of its concurrency key were running;
- more than 1% of the operations outside the restart windows were bad (the
  engine was not keeping up to begin with);
- the engine held more connections than `DATABASE_MAX_CONNS`, its DDL pool
  aside.

`compare` calls a track `IMPROVED` or `REGRESSED` only when the two sides'
ranges over their restarts do not overlap and the means differ by more than
the track's `NoiseFloorS`. With five restarts a side, two runs of the same
build have non-overlapping ranges 0.8% of the time. Anything else is
`NO CHANGE`.

## Tracks

The tracks are in `tracks.go`. Every tenant has a number of workflows, each
with one `GROUP_ROUND_ROBIN` concurrency strategy with `max_runs: 1`. Some of
them carry a standing backlog of 200 ms tasks on one key, so there are queued
slots for the scheduler to read back at every start. Probes use a key nothing
else uses, so only the engine's health delays them.

- `single-tenant-default`: one tenant, default pool settings. The control.
- `mt-many-equal`: many equal small tenants under per-tenant and shared
  connection caps.
- `mt-skewed`: one large tenant with thousands of strategies next to a few
  small ones, same caps. This is the track that feels work the scheduler does
  once per strategy at start.

The engine runs with its defaults apart from the track's pool settings, with
one exception: `SERVER_REASSIGN_UNACKED_TASKS=true`, so a task whose dispatch
is lost when the engine exits is reassigned rather than left to time out.

## On a pull request

A comment `/restart-harness` (or `/restart-harness <track>`) from someone with
write access, on a pull request from a branch of this repository, runs
`.github/workflows/restart-harness.yml`: the merge base and the head are built
and measured on one runner per track, and the comparison is posted back. The
workflow can also be started by hand for any two refs.

## Recalibrating

`NoiseFloorS` is how far the mean of two runs of the same build can differ. It
depends on the machine: after changing a track or the runner, run the same
build three times, compare the runs with each other and set the floor above
the largest difference between two valid runs. Results recorded with other
track settings are rejected as `INVALID`.

## Not covered

Database restarts, `SIGKILL`, two engines rolling over each other, pull
requests from forks, and latency percentiles.
