# Debug a Hatchet Run

These are instructions for an AI agent to diagnose why a Hatchet run failed, is stuck, or behaved unexpectedly. Follow each step in order.

## Step 1: Get Run State

```bash
hatchet runs get RUN_ID -o json -p HATCHET_PROFILE
```

Examine the JSON response:

- `.run.status` -- the overall run status (`QUEUED`, `RUNNING`, `COMPLETED`, `FAILED`, `CANCELLED`)
- `.run.displayName` -- the workflow name
- `.run.input` -- the input that was provided when the workflow was triggered
- `.tasks[]` -- each task in the run, with its own `status`, `displayName`, and `output` or `errorMessage`
- `.tasks[].startedAt` and `.tasks[].finishedAt` -- timing information

## Step 2: Get Event Log

```bash
hatchet runs events RUN_ID -o json -p HATCHET_PROFILE
```

The event log shows the full lifecycle of the run. Each event has:

- `eventType` -- what happened (e.g. `QUEUED`, `STARTED`, `FINISHED`, `FAILED`, `CANCELLED`)
- `message` -- human-readable description
- `taskDisplayName` -- which task the event belongs to
- `timestamp` -- when it happened

Read events in chronological order to understand the sequence of what happened.

## Step 3: Get Logs

```bash
hatchet runs logs RUN_ID -p HATCHET_PROFILE
```

This prints the application-level log output from the task code (e.g. print statements, logger calls). For multi-task (DAG) runs, logs from all tasks are merged and sorted by timestamp with task name prefixes.

## Diagnostic Cheat Sheet

### Task stuck in QUEUED (no STARTED event)

The task was never picked up by a worker. Likely causes:

- **No worker is running.** Start one with `hatchet worker dev -p HATCHET_PROFILE`.
- **Worker does not register this task type.** The task name in the workflow definition must match what the worker code registers. Check the worker startup logs.
- **Worker is connected to a different tenant/profile.** Verify the worker and the trigger use the same `-p` profile.

### Task FAILED

Check the logs output from Step 3 for a stack trace or error message. Common causes:

- Unhandled exception in the task code.
- Timeout exceeded.
- A dependency (database, API, etc.) was unreachable.

The events from Step 2 will show a `FAILED` event with an error message.

### Task CANCELLED

Check the events for a `CANCELLED` event. The message field indicates the cancellation source:

- Manual cancellation via the dashboard or CLI.
- Parent workflow cancellation (for DAG workflows).
- Timeout-based cancellation.

### Run COMPLETED but output is wrong

If the run completed but produced unexpected results:

1. Check `.tasks[].output` in the run state (Step 1) to see what each task returned.
2. Check the logs (Step 3) to trace the execution flow.
3. Check `.run.input` to verify the correct input was provided.

### Slow execution

Compare `startedAt` and `finishedAt` timestamps for each task in Step 1 to identify which task is the bottleneck. Check if there is a long gap between the run being triggered and the first task starting (indicates queuing delay, possibly due to worker capacity).
