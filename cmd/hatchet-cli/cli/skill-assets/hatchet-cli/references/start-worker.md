# Start a Hatchet Worker in Dev Mode

These are instructions for an AI agent to start a Hatchet worker using the CLI. Follow each step in order.

## Prerequisites

You need a `hatchet.yaml` file in the project root. If one does not exist, create it with the following structure:

```yaml
dev:
  runCmd: "python src/worker.py"
  files:
    - "**/*.py"
  reload: true
```

Adjust `runCmd` to match the project's language and entry point:

- Python: `poetry run python src/worker.py` or `python src/worker.py`
- TypeScript/Node: `npx ts-node src/worker.ts` or `npm run dev`
- Go: `go run ./cmd/worker`

The `files` list contains glob patterns for file watching. The `reload: true` setting enables automatic worker restart when watched files change.

## Start the Worker

Run the following command in a **background terminal** (the worker is a long-running process that must stay alive):

```bash
hatchet worker dev -p HATCHET_PROFILE
```

The worker will connect to Hatchet using the specified profile and begin listening for tasks.

## Important Notes

- The worker **must be running** before you trigger any workflows. If a workflow is triggered with no worker running, tasks will remain in QUEUED status indefinitely.
- When `reload: true` is set, the worker automatically restarts when any watched file changes. This means you can edit task code and the worker picks up changes without manual restart.
- To disable auto-reload, add `--no-reload`.
- To override the run command without editing `hatchet.yaml`, use `--run-cmd "your command here"`.
- If the worker fails to start, check that the profile exists (`hatchet profile list`) and that the Hatchet server is reachable.

## Optional: Pre-commands

You can add setup commands that run before the worker starts:

```yaml
dev:
  preCmds:
    - "poetry install"
    - "npm install"
  runCmd: "poetry run python src/worker.py"
  files:
    - "**/*.py"
  reload: true
```

These run once each time the worker starts (including on reload).
