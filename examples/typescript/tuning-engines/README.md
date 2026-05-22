# Tuning Engines governed AI workflow

This example shows how to call a Tuning Engines OpenAI-compatible endpoint from
a Hatchet TypeScript workflow. Hatchet owns durable execution, workers, retries,
and workflow state. Tuning Engines owns governed model access, policy checks,
budgets, audit logs, and runtime trace correlation.

## Environment

Set your Hatchet variables as usual, plus:

```bash
export TE_INFERENCE_KEY=sk-te-your-inference-key
export TE_MODEL=auto
```

## Files

- `workflow.ts` defines a workflow task that calls the governed AI endpoint.
- `worker.ts` starts a Hatchet worker for the workflow.
- `run.ts` triggers the workflow with a sample prompt.

Use the Hatchet run id or your own workflow correlation id as the Tuning Engines
`run_id` so usage, policy decisions, approvals, and traces can be correlated.
