---
name: embedded-hatchet
description: Test Hatchet features against an in-process (embedded mode) Hatchet instance built from this checkout, with its own throwaway embedded Postgres, optionally registering and running workflows. Use when asked to try out, verify, or demo engine/SDK changes without docker compose.
---

**📝 SELF-UPDATING DOCUMENT**: This skill automatically updates itself when inaccuracies are discovered or new patterns are learned. Always verify information against the actual codebase and update this file when needed.

# embedded-hatchet

Run a full Hatchet instance (engine + API + migrations) inside a single Go
process built from this repo checkout, backed by a throwaway
embedded Postgres that the same process spins up and tears down. Then
optionally register a worker and run workflows against it.

## How embedded mode works

- `hatchet.NewClient(hatchet.WithEmbeddedPostgres(databaseURL))` from
  `github.com/hatchet-dev/hatchet/sdks/go`, plus a blank import of
  `github.com/hatchet-dev/hatchet/embed`, starts everything in-process.
- Migrations and tenant/token seeding happen automatically on startup.
- Working examples live in `sdks/go/examples/embedded/`
  (`basic`, `worker`, `trigger`) — read `basic/main.go` first and model the
  test program on it.
- Options: `hatchet.WithEmbeddedGRPCPort(n)`, `WithEmbeddedAPIPort(n)`,
  `WithoutEmbeddedAPI()`, `WithEmbeddedLogLevel("debug")`. Defaults pick
  free ports.

## Steps

1. **Write the test program** in a temp directory (not inside the repo).
   `go mod init embedtest`, then in `go.mod` add
   `replace github.com/hatchet-dev/hatchet => <absolute path to this repo>`
   so the run exercises the local checkout, including any uncommitted
   changes being tested. Run `go mod tidy` after writing the code.

2. **Embed Postgres in the same program** with
   `github.com/fergusstrange/embedded-postgres` — no docker needed;
   binaries are downloaded once and cached under
   `~/.embedded-postgres-go`. Start it before the Hatchet client, stop it
   on exit:

   ```go
   pg := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
       Port(5499).
       Database("hatchet").
       StartParameters(map[string]string{"timezone": "UTC"}).
       RuntimePath(filepath.Join(os.TempDir(), "embedtest-pg")))
   if err := pg.Start(); err != nil { return err }
   defer pg.Stop()

   dbURL := "postgresql://postgres:postgres@localhost:5499/hatchet?sslmode=disable"
   client, err := hatchet.NewClient(hatchet.WithEmbeddedPostgres(dbURL))
   ```

   Pick a free port (avoid 5432/5433 which the compose stack may hold).
   The `timezone: UTC` start parameter is required — embedded Postgres
   inherits the system timezone and Hatchet refuses to start against a
   non-UTC database.

3. **Exercise the feature.** Adapt the program to what's being tested:
   - Just boot the instance and hit the REST API? Sleep and probe the API
     port with curl.
   - Run workflows? Define tasks with `client.NewStandaloneTask` /
     `client.NewWorkflow`, start a worker with `client.NewWorker` +
     `worker.Start()`, give it ~2s to register, then `task.Run(ctx, input)`
     and print the output.
   - Testing a specific engine/SDK change? Shape the workflow to hit that
     code path (retries, concurrency, DAG steps, etc.).

4. **Run it** with `go run .` from the temp dir. First run downloads
   Postgres binaries and applies migrations, so allow a minute. Report the
   actual output.

5. **Clean up**: the deferred `pg.Stop()` tears Postgres down with the
   process. Delete the temp dir (which removes the data dir too); don't
   leave built binaries behind.

## Notes

- Each concurrent run needs its own Postgres port and RuntimePath.
- If the process was killed and a stale embedded Postgres holds the port,
  `pkill -f embedtest-pg` and remove the RuntimePath dir.
- This is for local feature testing only — never point it at a shared or
  production database.
