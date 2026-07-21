# CEL-based display names for runs and steps — design spec

**Date:** 2026-07-21
**Branch:** `issue-4259`
**Issue:** #4259
**Supersedes:** the trigger-time literal `display_name` currently on this branch (commit `45b24240d` and follow-ups). This spec **replaces** that mechanism.

---

## 1. Goal

**What this enables.** Let workflow authors set a human-readable label on a run — and on individual steps of a DAG — via a CEL expression declared in the workflow/task **definition**, evaluated at trigger time against the run input. Because the expression lives in the definition (not the trigger call), every trigger source — manual, event, and cron — gets meaningful names automatically, and each step of a DAG can be named independently. This closes #4259 (fanned-out child/bulk runs all rendering as an identical `<name>-<timestamp>` in the dashboard) and delivers the reviewer's ask (per-step names, consistent with other per-step config).

**Success criteria.**
- A workflow-level `displayName` CEL expression is stored on the workflow version, evaluated at DAG creation, and written to `v1_dag.display_name`; the runs list shows the evaluated name.
- A per-step `displayName` CEL expression is stored on the step, evaluated at task insert (root **and** lazily-created downstream tasks), and written to `v1_task.display_name`.
- A single-step workflow (which produces no `v1_dag` row) is named by its **step-level** `displayName`, falling back to the **workflow-level** `displayName` when no step-level one is set.
- Manual, event, and cron triggers all produce evaluated run names. Per-step names need no per-trigger code (they resolve in the common `insertTasks`); the run-level name is wired through both trigger-prep paths (`prepareTriggerFromWorkflowNames` for manual/cron/scheduled, `prepareTriggerFromEvents` for events).
- A malformed expression is rejected at **registration** (workflow put), not at run time.
- A runtime evaluation error (missing key, non-string result, empty result) falls back to the generated `<readableId|workflowName>-<timestamp>` name and **never fails the run**.
- The trigger-time `display_name` field and every SDK `RunOpts.displayName` surface added on this branch are removed.

**Non-goals.**
- No `{{ }}` string-template interpolation — the whole string is a pure CEL expression (consistent with concurrency / rate-limit expressions).
- No access to parent-step outputs in a name expression — input-only environment.
- No trigger-time literal override — runtime values are supplied by referencing the run input (`displayName: "input.displayName"`).
- No UI trigger-form field and no frontend changes — the read side already returns `displayName`.

---

## 2. Folder structure & layer placement

No new packages. Changes land in existing files across the contract, engine, and SDK layers.

```txt
api-contracts/v1/
  shared/trigger.proto                         MODIFIED — remove display_name (#11)
  workflows.proto                              MODIFIED — remove display_name from TriggerWorkflowRunRequest (#6);
                                                          add display_name to CreateTaskOpts (#16) & CreateWorkflowVersionRequest (#15)
  openapi/.../v1/workflow_run.yaml             MODIFIED — revert trigger displayName addition

sql/schema/v0.sql                              MODIFIED — add "displayName" TEXT to "Step" and "WorkflowVersion"
cmd/hatchet-migrate/migrate/migrations/
  <ts>_v1_0_126.sql                            NEW — goose up/down: ADD/DROP COLUMN "displayName" on "Step" & "WorkflowVersion"

pkg/repository/
  sqlcv1/ (regenerated)                        MODIFIED — CreateStepParams, ListStepsByIdsRow,
                                                          CreateWorkflowVersionParams, ListWorkflowsByNamesRow gain DisplayName
  sqlcv1/*.sql                                 MODIFIED — CreateStep, CreateWorkflowVersion, ListStepsByIds, ListWorkflowsByNames
  workflow.go                                  MODIFIED — CreateWorkflowVersionOpts + CreateStepOpts gain DisplayName (validated);
                                                          write to CreateWorkflowVersionParams & CreateStepParams
  trigger.go                                   MODIFIED — triggerTuple carries the workflow-level EXPRESSION;
                                                          createDAGs evaluates it; revert trigger-side literal threading
  task.go                                      MODIFIED — insertTasks evaluates stepConfig.DisplayName expression
  input.go                                     UNCHANGED (referenced: TaskInput.Input is the run input)

internal/services/admin/v1/server.go          MODIFIED — revert admin-path display_name normalization
api/v1/server/handlers/v1/workflow-runs/trigger.go   MODIFIED — revert REST trigger display_name field

internal/cel/cel.go                            UNCHANGED (reuse ParseAndEvalWorkflowString / workflowStrEnv)
pkg/validator/validator.go                     UNCHANGED (reuse "celworkflowrunstr" tag)
pkg/repository/trigger.go NormalizeDisplayName  KEPT — now applied to the evaluated output, not the raw trigger arg

sdks/typescript/src/v1/{task.ts,declaration.ts}      MODIFIED — displayName moves to CreateBaseTaskOpts + CreateBaseWorkflowOpts; remove from RunOpts
sdks/python/hatchet_sdk/runnables/workflow.py        MODIFIED — display_name moves to workflow/task definition config; remove from run/run_no_wait/aio
sdks/go/workflow.go, sdks/go/durable_child.go        MODIFIED — remove WithRunDisplayName; add WithDisplayName (TaskOption) + WithWorkflowDisplayName (WorkflowOption)
sdks/ruby/src/lib/hatchet/...                        MODIFIED — display_name moves to step/workflow config; remove from trigger_options
frontend/docs/pages/v1/run-names.mdx                 MODIFIED — rewrite guide for CEL display-name expressions

CHANGELOG.md + sdks/{python,typescript,ruby}/CHANGELOG.md   MODIFIED — update entries to describe the CEL mechanism
```

**Layer placement rationale.**
- `Step.displayName` / `WorkflowVersion.displayName` are new nullable scalar columns, mirroring existing per-step scalar config (`readableId`, `timeout`, `scheduleTimeout`) and per-version config (`defaultPriority`, `inputJsonSchema`). They belong in the definition tables because a display-name expression is workflow configuration, not run state.
- Evaluation lives in the repository trigger path (`insertTasks`, `createDAGs`) because that is the only place with both the stored expression **and** the run input in scope, and it is where `v1_task.display_name` / `v1_dag.display_name` are written today.

**Migration note.** `task migrate:new-version` (goose) generates a timestamped migration; add the two `ALTER TABLE ... ADD COLUMN "displayName" TEXT` statements (with matching `-- +goose Down` `DROP COLUMN`s). Mirror the change into `sql/schema/v0.sql` so a fresh schema build matches, then re-run sqlc generation so the four query rows/params above pick up the column.

---

## 3. Per-file changes

### `api-contracts/v1/shared/trigger.proto`
- **What changes:** remove `optional string display_name = 11;` from `TriggerWorkflowRequest`. Field number 11 is retired (do not reuse).
- **Edge cases:** none — pure removal; regeneration drops it from all SDK stubs.

### `api-contracts/v1/workflows.proto`
- **What changes:**
  - `TriggerWorkflowRunRequest`: remove `optional string display_name = 6;`.
  - `CreateTaskOpts`: add `optional string display_name = 16; // (optional) a CEL expression for the task's display name`.
  - `CreateWorkflowVersionRequest`: add `optional string display_name = 15; // (optional) a CEL expression for the run's display name`.
- **Edge cases:** field numbers 16 / 15 are the next free numbers in each message (verified against current `.proto`). Retired trigger numbers (11 / 6) must not be reused.

### `sql/schema/v0.sql` + goose migration
- **Purpose:** persist the two expressions.
- **What changes:** `"Step"` gains `"displayName" TEXT` (nullable, no default); `"WorkflowVersion"` gains `"displayName" TEXT` (nullable, no default). Migration `up` adds both columns; `down` drops both.
- **Edge cases:** nullable with no default → existing rows read `NULL` → generated-name fallback (unchanged behavior for every already-registered workflow). Adding a nullable column with no default is a non-blocking DDL on Postgres.

### `pkg/repository/sqlcv1/*.sql` (+ regenerated `*.sql.go`)
- **`CreateStep`:** add `"displayName"` to the column list and `sqlc.narg('displayName')::text` to `VALUES`. Regen adds `DisplayName pgtype.Text` to `CreateStepParams`.
- **`ListStepsByIds`:** add `steps."displayName"` to the SELECT. Regen adds `DisplayName pgtype.Text` to `ListStepsByIdsRow` (this row is `stepConfig` in `insertTasks`).
- **`CreateWorkflowVersion`:** add `"displayName"` to the column list and `sqlc.narg('displayName')::text` to `VALUES`. Regen adds `DisplayName pgtype.Text` to `CreateWorkflowVersionParams`.
- **`ListWorkflowsByNames`:** add `workflowVersions."displayName" AS "displayName"` to the SELECT. Regen adds `DisplayName pgtype.Text` to `ListWorkflowsByNamesRow`. This row feeds the **manual, cron, and scheduled** trigger paths (all three build `WorkflowNameTriggerOpts` → `TriggerFromWorkflowNames` → `prepareTriggerFromWorkflowNames`; verified `cron_v1.go:43`, `schedule_workflow_v1.go:44`).
- **`ListWorkflowsForEvents`:** add `workflowVersions."displayName"` to the `latest_versions` CTE **and** `latest_versions."displayName"` to the outer SELECT. Regen adds `DisplayName pgtype.Text` to `ListWorkflowsForEventsRow`. **This is the fix for the event-trigger path** — without it, event-triggered DAG runs have no workflow-level expression to evaluate and silently fall back to the generated name.
- **Edge cases:** `ListWorkflowsByNames` results are cached in `tenantIdWorkflowNameCache` (5s TTL, keyed by row). The expression is stable per workflow version, so caching is safe; a changed expression cuts a new workflow version (new checksum) and thus a new cached row.

### `pkg/repository/workflow.go`
- **`CreateWorkflowVersionOpts`** (struct at line ~27): add `DisplayName *string \`json:"displayName,omitempty" validate:"omitnil,celworkflowrunstr"\``.
- **`CreateStepOpts`** (struct at line ~72): add `DisplayName *string \`json:"displayName,omitempty" validate:"omitnil,celworkflowrunstr"\``.
- **`createWorkflowVersionTxs`** (line ~415): set `createParams.DisplayName = pgtype.Text{...}` from `opts.DisplayName` when non-nil.
- **step insert** (`createStepParams`, line ~736): set `createStepParams.DisplayName` from `stepOpts.DisplayName` when non-nil.
- **`checksumV1`** (line ~1268): no code change needed — it hashes the marshaled opts, so the new fields participate in the checksum automatically; a changed expression produces a new workflow version. Confirm the checksum test is updated to reflect the new field (see Tests).
- **Edge cases:** `omitnil` means an unset expression is skipped entirely by the validator; only a present-but-uncompilable expression fails registration. The `celworkflowrunstr` validator compiles against `workflowStrEnv`, so an expression referencing `parents` (not in that env) fails at registration — intended.
- **Error handling:** a compile failure surfaces as a validation error from `PutWorkflowVersion` → returned to the SDK/gRPC caller as an `expected`/terminal registration error (same path as an invalid concurrency expression today). No new error class; reuse the validator's aggregated error.

### `pkg/repository/trigger.go`
- **`triggerTuple.displayName`** changes meaning: it now carries the **workflow-level CEL expression** (the run-name expression), not a caller-supplied literal. It must be populated at **both** `triggerTuple{}` construction sites (there are exactly two, verified):
  - **`prepareTriggerFromWorkflowNames`** (site ~line 2389): set `displayName` from `ListWorkflowsByNamesRow.DisplayName`. Covers manual, cron, and scheduled triggers.
  - **`prepareTriggerFromEvents`** (site ~line 2281): set `displayName` from `ListWorkflowsForEventsRow.DisplayName`. **Covers event triggers — previously missing.** Without this, event-triggered DAG runs never get a workflow-level name.
  - Remove all population from `TriggerTaskData`/`opt.DisplayName` (that trigger-time literal path is deleted).
- **`createDAGs`** (line ~1447): replace the literal branch with evaluation. For each `opt`, if the workflow-level expression is present, evaluate `r.celParser.ParseAndEvalWorkflowString(expr, in)` where `in = cel.NewInput(cel.WithInput(runInput), cel.WithAdditionalMetadata(meta), cel.WithWorkflowRunID(opt.ExternalId))`; run the result through `NormalizeDisplayName`; on any error / nil / empty, use the generated `fmt.Sprintf("%s-%d", opt.WorkflowName, unix)`.
  - **Input shape (verified — `trigger.go:1195`):** `createDAGOpts.Input` is the **raw run input** `[]byte` (`tuple.input`), **not** a wrapped `TaskInput`. Unmarshal it directly into `map[string]interface{}` and pass that as `runInput`. (Do **not** write `opt.Input.input` — there is no wrapper here. Only the *task* path wraps via `r.newTaskInput(...)` at `trigger.go:1054`, which is why `insertTasks` uses `task.Input.Input` but `createDAGs` uses the bare map.)
  - `meta` = unmarshal `opt.AdditionalMetadata` (raw `[]byte`); if nil/empty pass an empty map so `additional_metadata` is always bound.
- **`createDAGOpts`**: replace `DisplayName *string` (resolved literal) with the workflow-level expression string carried from the tuple. (Field can keep the name; its semantics change to "expression".)
- **`NormalizeDisplayName`** (kept): still trims, empties→nil, truncates to 255 runes — now applied to the **evaluated** output.
- **Single-task (non-DAG) branch** (site ~line 1070): set the single-task `CreateTaskOpts.WorkflowDisplayName = tuple.displayName` (the workflow-level expression) so a single-step run is named by the workflow-level expression when no step-level one is set (see `task.go` / finding #3 fix). Do **not** set it on DAG task opts.
- **Remove:** `NewTriggerTaskData`'s `DisplayName: NormalizeDisplayName(req.DisplayName)`, the `TriggerTaskData.DisplayName` field, and the single-task `opt.DisplayName = tuple.displayName` literal assignment.
- **Edge cases:** a malformed/empty `opt.Input` yields an empty `input` map, so `input.x` references error → fallback. Event-path input (`opt.Data` → `tuple.input`) is the same raw shape as the manual path, so evaluation is uniform across trigger sources.
- **Error handling:** all evaluation errors are swallowed to the generated fallback (log at debug/warn, do not propagate). A cosmetic label must never fail DAG creation — `terminal` is explicitly wrong here.

### `pkg/repository/task.go`
- **`insertTasks`** (display-name branch, line ~1980): replace the literal `task.DisplayName` branch with expression evaluation using this precedence (**step-level → workflow-level → generated**):
  - `in := cel.NewInput(cel.WithInput(task.Input.Input), cel.WithAdditionalMetadata(<unmarshaled task.AdditionalMetadata>), cel.WithWorkflowRunID(task.WorkflowRunId))`.
  - If `stepConfig.DisplayName` (the new `pgtype.Text` on `ListStepsByIdsRow`) is `Valid` and non-empty → evaluate it via `r.celParser.ParseAndEvalWorkflowString`.
  - Else if `task.WorkflowDisplayName` is set (only populated for single-task runs — see below) → evaluate it. **This is the finding-#3 fix**: a single-step run is named by the workflow-level expression when it has no step-level one, instead of silently ignoring it.
  - Else / on any eval error / empty → generated `fmt.Sprintf("%s-%d", stepConfig.ReadableId.String, unix)`.
  - Normalize any evaluated result through `NormalizeDisplayName`.
- **`CreateTaskOpts`:** remove the resolved-literal `DisplayName *string`; add `WorkflowDisplayName *string` — the **workflow-level expression**, set **only** on the single-task (non-DAG) branch in `triggerWorkflows` (DAG step opts leave it nil, so DAG step names never inherit the run-level expression). The per-step expression itself is read from `stepConfig`, not carried on the opt.
- **CEL parser access:** reuse `r.celParser` — `sharedRepository` already holds a `*cel.CELParser` (`shared.go:44`) and `insertTasks` already calls it for concurrency-key evaluation (`task.go:2123`), so no new wiring is needed. Do not construct a new parser per call.
- **Edge cases:**
  - **Root vs downstream tasks:** both funnel through `insertTasks`; `task.Input.Input` is the **run input** in both cases (verified: `TaskInput.Input` is the original run input; parent outputs live in `TaskInput.TriggerData`, which this feature intentionally ignores). So a step name evaluates identically regardless of when the task is created.
  - **Single-step workflow:** `isDag == false`, the one step's task is the run. It is named by its step-level expression if set, otherwise by the workflow-level expression (`task.WorkflowDisplayName`), otherwise generated. No `v1_dag` row is involved.
  - **`task.WorkflowRunId`** is the run's UUID — correct binding for `workflow_run_id`.
  - `task.AdditionalMetadata` may be empty → pass empty map.
- **Error handling:** identical to `createDAGs` — swallow to generated fallback, never fail task insert.

### `internal/services/admin/v1/server.go` + `api/v1/server/handlers/v1/workflow-runs/trigger.go`
- **What changes:** revert the display_name normalization / field-plumbing added on this branch. These paths no longer see a trigger-time display name.
- **Edge cases:** none — pure revert; confirm no other caller reads the removed request field.

### SDKs (surface moves from run-call to definition)

For each SDK, mirror exactly how the existing **concurrency** expression is declared on the task/workflow definition and threaded into the proto; `displayName` is a sibling CEL-string field.

- **TypeScript** — add `displayName?: string` to `CreateBaseTaskOpts` (`sdks/typescript/src/v1/task.ts`) and to `CreateBaseWorkflowOpts` (`sdks/typescript/src/v1/declaration.ts`); thread into `CreateTaskOpts.displayName` / `CreateWorkflowVersionRequest.displayName` at registration. Remove `displayName` from `RunOpts` (`declaration.ts:83`) and from `worker/context.ts` (child spawn).
- **Python** — add `display_name: str | None = None` to the workflow definition config and the task definition config (where `concurrency` is set in `runnables/workflow.py`); thread into the two proto messages. Remove `display_name` from `run` / `run_no_wait` / `aio_run` / `run_many` item builders and from `types/trigger.py`.
- **Go** — remove `WithRunDisplayName` (and its `RunOpts.DisplayName`). Add `WithDisplayName(expr string) TaskOption` (mirror `WithConcurrency`) and `WithWorkflowDisplayName(expr string) WorkflowOption` (mirror `WithWorkflowConcurrency`); set the expression on the step/workflow definition opts. Update `durable_child.go` to drop the run-time display name.
- **Ruby** — move `display_name` from `trigger_options.rb` to the step/workflow definition config; thread into the definition request. Update `.rbs` signatures.
- **Edge cases (all SDKs):** the value is an opaque CEL string passed through unchanged; no client-side validation or truncation (the server validates at registration and truncates the evaluated output). Omitting it reproduces today's generated-name behavior.

### Docs — `frontend/docs/pages/v1/run-names.mdx`
- Rewrite from "literal name at trigger time" to "CEL display-name expressions on the workflow and its steps," covering: pure-CEL syntax (quote literals: `"'Acme Corp'"`), reading input (`"input.customerName"`), the `has()` guard for optional keys (`"has(input.name) ? input.name : 'run'"`), workflow-level vs per-step, the single-step-workflow note, and event/cron coverage. Keep the cross-links from `additional-metadata.mdx` and `child-spawning.mdx`.

---

## 4. Flow

### Registration (workflow put)
1. SDK builds `CreateWorkflowVersionRequest` with workflow-level `display_name` and per-task `CreateTaskOpts.display_name`.
2. Engine maps into `CreateWorkflowVersionOpts.DisplayName` and each `CreateStepOpts.DisplayName`.
3. `PutWorkflowVersion` runs the validator: `celworkflowrunstr` compiles each expression against `workflowStrEnv`. A compile failure returns a validation error and **no version is created**.
4. `checksumV1` hashes the opts (including the expressions). If the checksum matches the current version, no new version is created; otherwise a new `WorkflowVersion` row is inserted with `displayName`, and each `Step` row is inserted with `displayName`.

### Manual / event / cron trigger — DAG run (2+ steps)
1. `prepareTriggerFromWorkflowNames` loads `ListWorkflowsByNamesRow` (now including `DisplayName`) → `triggerTuple.displayName = <workflow-level expression>`.
2. `triggerWorkflows` classifies `isDag = len(steps) > 1` → true; builds `createDAGOpts` carrying the expression, the run input, additional metadata, and the run external id.
3. `createDAGs` evaluates the expression against `{input, additional_metadata, workflow_run_id}`; normalizes; writes `v1_dag.display_name` (or generated fallback).
4. Root step tasks are inserted via `insertTasks`; each evaluates its own `stepConfig.DisplayName` against the same run input → `v1_task.display_name` (or generated fallback).
5. As parents complete, downstream tasks are created lazily (match conditions) and pass through `insertTasks` again — same per-step evaluation, same run input.

### Manual / event / cron trigger — single-task run (1 step)
1. `isDag = false`; the step's task is the run. The single-task branch sets `CreateTaskOpts.WorkflowDisplayName` from the tuple's workflow-level expression.
2. `insertTasks` resolves the name by precedence: step-level expression → workflow-level expression (`task.WorkflowDisplayName`) → generated `<readableId>-<ts>`. Result → `v1_task.display_name`. (No `v1_dag` row exists, so the workflow-level expression is applied to the task rather than a DAG row.)

### Unhappy paths
- **Compile error** → caught at registration; version not created (loud, `expected`).
- **Runtime eval error / non-string / empty result** → generated `<readableId|workflowName>-<timestamp>` fallback; run proceeds (silent, logged at debug/warn).
- **Missing input key** (`input.x` absent) → CEL "no such key" error → fallback. Authors use `has()` to avoid this.
- **Over-long result** → truncated to 255 runes by `NormalizeDisplayName` (never rejected, rune-safe).

### Concurrency / ordering
- The resolved display name is **not** part of any idempotency/dedup key (unchanged from the current branch's invariant). On durable replay or child re-trigger, the run is matched on its unchanged idempotency key and the first-stored name is reused; a re-evaluated name is ignored — no `NonDeterminismError`. Preserve this: the eval output feeds only the `display_name` write, never a key.

---

## 5. Tests

### Engine — unit
- `it("compiles and stores a workflow-level display name expression")` — `PutWorkflowVersion` with a valid workflow-level expression persists `WorkflowVersion.displayName`.
- `it("compiles and stores a per-step display name expression")` — persists `Step.displayName`.
- `it("rejects an uncompilable display name expression at registration")` — invalid CEL → validation error, no version created.
- `it("includes the display name expression in the workflow checksum")` — changing only the expression yields a new checksum / new version (update existing `workflow_checksum_test.go`).
- `it("normalizes an evaluated name: trims, empties→fallback, truncates to 255 runes")` — `NormalizeDisplayName` unit coverage retained.

### Engine — integration (`trigger_display_name_integration_test.go`, rewritten)
- `it("names a single-task run from its step expression")` — `input.name` → `v1_task.display_name`.
- `it("names a single-task run from the workflow-level expression when no step expression is set")` — finding-#3 fix: workflow-level expr → the single task's `v1_task.display_name` (not the generated fallback).
- `it("prefers the step-level expression over the workflow-level one on a single-task run")` — precedence assertion.
- `it("names a DAG run from the workflow-level expression against the raw run input")` — asserts `input.name` (top-level, un-nested) resolves in `createDAGs` → `v1_dag.display_name`. Guards the finding-#1 raw-input shape: a DAG triggered with ordinary input must NOT fall back.
- `it("names each DAG step from its per-step expression")` — distinct `v1_task.display_name` per step, root and downstream.
- `it("falls back to the generated name when no expression is set")`.
- `it("falls back to the generated name when the expression references a missing key")` — no run failure.
- `it("falls back when the expression evaluates to a non-string")`.
- `it("names an event-triggered DAG run from the workflow-level expression")` — finding-#2 fix: exercises `prepareTriggerFromEvents` + `ListWorkflowsForEvents`; asserts `v1_dag.display_name` is evaluated, not generated.
- `it("names a cron/scheduled-triggered DAG run from the workflow-level expression")` — exercises the `WorkflowNameTriggerOpts` path; asserts evaluated `v1_dag.display_name`.
- `it("gives distinct names across a fanned-out run_many batch")` — the original #4259 scenario, names derived from per-item input.
- `it("reuses the first-spawn name on durable replay / child re-trigger")` — display name excluded from idempotency key ("first-name-wins").
- **Fixtures:** a single-task workflow with (a) a step expression and (b) only a workflow-level expression; a 2-step DAG with workflow-level + per-step expressions; the DAG bound to both an event trigger ref and a cron ref, triggered through each path.

### Engine — error paths
- `it("does not fail DAG creation when the workflow-level expression throws")` — `createDAGs` returns the DAG with the generated fallback, no error propagated.
- `it("does not fail task insert when a step expression throws")` — `insertTasks` succeeds with the generated fallback.

### SDK — request-builder unit (per SDK)
- `it("threads displayName from the workflow definition into CreateWorkflowVersionRequest.display_name")`.
- `it("threads displayName from a task definition into CreateTaskOpts.display_name")`.
- `it("no longer accepts displayName on the run call")` — the removed surface is gone (compile-level for typed SDKs; explicit assertion for Ruby/Python).

### Real-world scenarios considered
- Large `run_many` fan-out (names must be per-item, derived from each item's input) → covered by the batch test.
- Event and cron triggers with no author-side trigger code → covered.
- Author forgets `has()` on an optional key → graceful fallback, run still executes → covered.
- Durable replay mid-run → first-name-wins, no non-determinism → covered.
- Re-registration changing only the expression → new version, old runs unaffected → covered by checksum test.

### Explicitly out of scope for tests
- Snapshot tests on proto shapes; tests that mock `insertTasks`/`createDAGs`; any `{{ }}` template behavior (not implemented).

---

## 6. Skills & docs the implementer may need

- **`tdd`** — red-green-refactor for the engine changes; write the eval/fallback tests first.
- **`build-tui-view`** — not needed (no TUI change).
- **`internal/cel/cel.go`** — the parser API: `ParseAndEvalWorkflowString`, `NewInput`, `WithInput/WithAdditionalMetadata/WithWorkflowRunID`, `workflowStrEnv` variable set.
- **`pkg/validator/validator.go`** — the `celworkflowrunstr` tag to attach to the new opts fields.
- **Existing concurrency expression path** (`CreateConcurrencyOpts.Expression`, `WithConcurrency`, Python `ConcurrencyExpression`) — the exact precedent to mirror for storage, validation, proto threading, and every SDK surface.
- **goose migrations** — `Taskfile.yaml` targets `migrate:new-version` / `migrate`; `hack/dev/migrate.sh` applies them; keep `sql/schema/v0.sql` in sync and re-run sqlc.
- **`CONTRIBUTING.md` / `contributing/`** — proto + OpenAPI regeneration steps (pinned generator versions).

---

## 7. Explicitly NOT doing

- **`{{ }}` string-template interpolation.** Decided: pure CEL only, consistent with concurrency/rate-limit. Literals are quoted CEL. *Why:* one evaluation model, no new parser, matches the codebase.
- **Trigger-time literal `display_name` override.** Removed entirely. *Why:* the reviewer prefers a single CEL mechanism; runtime values go through the input (`displayName: "input.displayName"`).
- **Parent-step outputs in name expressions.** Input-only env. *Why:* parent outputs are empty for root steps and the DAG run, so the same field would work on some steps and fall back on others — inconsistent; deferrable to a follow-up once there's demand.
- **Rejecting workflow-level `displayName` on single-step workflows at registration.** Considered (per adversarial review) but not chosen. *Why:* instead of erroring, a single-step run applies the workflow-level expression to its task when no step-level one is set (precedence: step → workflow → generated, in `insertTasks` via `CreateTaskOpts.WorkflowDisplayName`). This removes the "accepted-but-silently-ignored" footgun without a registration-time rejection that would break if a workflow's step count changes across versions.
- **New index / search on `display_name`.** *Why:* metadata filtering already covers search; names are display-only.
- **Frontend / UI changes.** *Why:* the read side already returns `displayName`; this is backend + SDK + docs only.
- **Surfacing eval failures in the UI (option B from design).** *Why:* silent fallback for the initial scope; add visibility later if it becomes a support pain.

---

## 8. Cleanup checklist — every file the old PR touched

The trigger-time PR (`origin/main...issue-4259`) changed **52 files**. Each is classified below so the pivot removes exactly what's no longer needed. `NormalizeDisplayName` (in `pkg/repository/trigger.go`) is the one helper that **stays** (re-applied to the evaluated output).

**DELETE (net-new, trigger-only; replace with definition-level tests per §5):**
- `internal/services/admin/v1/server_test.go`
- `pkg/client/admin_display_name_test.go`
- `sdks/go/display_name_test.go`
- `sdks/python/tests/unit/test_trigger_display_name.py`
- `sdks/ruby/src/spec/hatchet/clients/grpc/admin_spec.rb`
- `sdks/ruby/src/spec/hatchet/trigger_options_spec.rb`

**REWRITE (net-new, keep the file, repurpose for CEL):**
- `frontend/docs/pages/v1/run-names.mdx` — trigger-literal guide → CEL display-name guide.
- `pkg/repository/trigger_display_name_integration_test.go` — rewrite as the CEL integration suite (§5).

**REVERT (pre-existing files; back out the old-PR lines entirely):**
- `api-contracts/openapi/components/schemas/v1/workflow_run.yaml` (trigger `displayName`)
- `api-contracts/v1/shared/trigger.proto` (remove `#11`)
- `api/v1/server/handlers/v1/workflow-runs/trigger.go` (REST field)
- `internal/services/admin/v1/server.go` (admin normalization)
- `pkg/client/admin.go` — **(was missing from the earlier revert list)** the low-level Go engine client `WithDisplayName`
- `pkg/worker/context.go` — **(was missing)** Go worker child-spawn display name
- `pkg/repository/durable_events_test.go` — **(was missing)** remove the added display-name assertions
- `pkg/repository/trigger_test.go` — **(was missing)** remove the added display-name assertions
- `sdks/go/client.go`, `sdks/go/durable_child.go` (run/child display name)
- `sdks/python/hatchet_sdk/clients/admin.py`, `sdks/python/hatchet_sdk/types/trigger.py`
- `sdks/ruby/src/lib/hatchet/clients/grpc/admin.rb`, `.../trigger_options.rb`, `.../sig/hatchet/trigger_options.rbs`
- `sdks/typescript/src/v1/client/admin.ts`, `.../client/admin.test.ts`, `.../client/worker/context.ts`

**REWORK (changed by the new design — remove the trigger surface, add the definition surface):**
- `api-contracts/v1/workflows.proto` (remove `#6`; add `CreateTaskOpts #16` + `CreateWorkflowVersionRequest #15`)
- `pkg/repository/task.go`, `pkg/repository/trigger.go`, **`pkg/repository/workflow.go`** (new registration write — not in the old PR)
- `sdks/go/workflow.go` (drop `WithRunDisplayName`; add `WithDisplayName`/`WithWorkflowDisplayName`)
- `sdks/python/hatchet_sdk/runnables/workflow.py`, `sdks/typescript/src/v1/{declaration.ts,task.ts}`, Ruby step/workflow config
- `CHANGELOG.md`, `sdks/{python,typescript,ruby}/CHANGELOG.md` (rewrite the entry text)

**REGEN — do NOT hand-edit; these revert-and-re-add automatically when protos/OpenAPI are regenerated:**
- `api/v1/server/oas/gen/openapi.gen.go`, `frontend/app/src/lib/api/generated/data-contracts.ts`, `pkg/client/rest/gen.go`
- `internal/services/shared/proto/v1/{trigger,workflows}.pb.go`
- `sdks/python/hatchet_sdk/contracts/v1/shared/trigger_pb2.py(.pyi)`, `.../workflows_pb2.py(.pyi)`
- `sdks/ruby/src/lib/hatchet/contracts/v1/shared/trigger_pb.rb`, `.../workflows_pb.rb`
- `sdks/typescript/src/protoc/v1/shared/trigger.ts`, `.../workflows.ts`

**KEEP (no change needed):**
- `frontend/docs/pages/v1/_meta.js` — the `run-names` nav entry stays (the guide is rewritten, not removed).
- `frontend/docs/pages/v1/additional-metadata.mdx`, `.../child-spawning.mdx` — cross-links stay valid; adjust wording only if the guide's framing changes.

## 9. Verification statement

Every fact above is verified against the code on branch `issue-4259` — the CEL parser API, the `celworkflowrunstr` validator, the `Step` / `WorkflowVersion` schema and their `CreateStep` / `CreateWorkflowVersion` / `ListStepsByIds` / `ListWorkflowsByNames` / `ListWorkflowsForEvents` queries, the `isDag = len(steps) > 1` branch, the `insertTasks` / `createDAGs` write sites, and the current trigger-time surfaces to revert. Verified during adversarial-review follow-up: (1) `createDAGOpts.Input` is the **raw** run input `[]byte` (`trigger.go:1195`), whereas DAG *tasks* are wrapped via `r.newTaskInput` (`trigger.go:1054`) — so `createDAGs` unmarshals the bare map while `insertTasks` reads `task.Input.Input`; (2) there are exactly **two** `triggerTuple{}` sites — `prepareTriggerFromEvents` (`trigger.go:2281`, event path) and `prepareTriggerFromWorkflowNames` (`trigger.go:2389`) — and cron/scheduled triggers route through the latter via `WorkflowNameTriggerOpts` (`cron_v1.go:43`, `schedule_workflow_v1.go:44`); (3) the single-step run-name fallback is applied in `insertTasks` via `CreateTaskOpts.WorkflowDisplayName`. No open questions, no unverified assumptions, no TBDs.
