// Wrapper usage. `createWorkflowBuilder` is a reusable factory marked with the
// `@hatchet-workflow` JSDoc tag; the DAG is defined at the USAGE site below and
// renders on `ordersDag`. The factory body uses a dynamic name (`stub.name`)
// with explicit generics — the case that previously hid it from the visualizer.

type JsonObject = Record<string, unknown>;
interface WorkflowOutputSentinel {}

interface WorkflowStub<TInput> {
  name: string;
  version?: string;
}

declare function getClient(): {
  workflow<I extends JsonObject, O>(opts: { name: string; version?: string }): any;
};

/**
 * Reusable wrapper that builds a bare workflow from a stub. Callers attach the
 * task graph to the returned object (see usage below).
 *
 * @hatchet-workflow
 */
export function createWorkflowBuilder<TInput extends JsonObject>(
  stub: WorkflowStub<TInput>,
) {
  return getClient().workflow<TInput & JsonObject, WorkflowOutputSentinel>({
    name: stub.name,
    ...(stub.version ? { version: stub.version } : {}),
  });
}

// ── Usage: the DAG shape is defined here and renders on `ordersDag` ──────────
const ordersDag = createWorkflowBuilder({ name: 'orders-dag' });

const start = ordersDag.task({ name: 'start' });
const branchA = ordersDag.task({ name: 'branch-a', parents: [start] });
const branchB = ordersDag.task({ name: 'branch-b', parents: [start] });
const join = ordersDag.task({ name: 'join', parents: [branchA, branchB] });
