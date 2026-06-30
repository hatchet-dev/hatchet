// Real Hatchet SDK (v1) usage. Detection is type-driven: anything whose type
// resolves — through Promise/await, customer aliases, and inference — to an SDK
// workflow base type (WorkflowDeclaration / TaskWorkflowDeclaration /
// BaseWorkflowDeclaration) gets the DAG lens, however many wrappers deep.
import { HatchetClient } from '@hatchet-dev/typescript-sdk/v1';
import type { WorkflowDeclaration } from '@hatchet-dev/typescript-sdk/v1';

const hatchet = HatchetClient.init();

type DagInput = {
  Message: string;
};

// A customer-style alias over the SDK type — the kind of indirection that
// defeats syntactic detection but not type resolution.
type DurableWorkflow = WorkflowDeclaration<DagInput, {}, {}>;

// Factory returns the aliased SDK workflow type → lens on the function.
function createWorkflowBuilder(name: string): DurableWorkflow {
  return hatchet.workflow<DagInput>({ name });
}

// Usage: `wf` is typed (via the alias) as a workflow → lens here too; the DAG is
// the durableTask graph attached below.
export function createWrapperWorkflow() {
  const wf = createWorkflowBuilder('wrapper-example');

  const fetchData = wf.durableTask({
    name: 'fetch-data',
    fn: async (input) => ({ message: input.Message }),
  });

  const analyze = wf.durableTask({
    name: 'analyze',
    parents: [fetchData],
    fn: async () => ({ analyzed: true }),
  });

  wf.task({
    name: 'report',
    parents: [fetchData, analyze],
    fn: async () => ({ done: true }),
  });

  return wf;
}

// Direct + inferred — no wrapper, no annotation.
export const directWorkflow = hatchet.workflow<DagInput>({ name: 'direct-example' });
