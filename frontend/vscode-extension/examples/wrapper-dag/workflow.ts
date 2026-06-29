// `createWorkflowBuilder` is an `@hatchet-workflow` factory: it builds the
// workflow AND its task DAG internally (dynamic name `stub.name` + generics) and
// returns it. The CodeLens lands on the *usage* (`builtWorkflow` at the bottom),
// and clicking it renders the DAG defined inside the factory:
//
//   step-1 ─┐
//           ├─► step-3 ─► step-5 ─┐
//   step-2 ─┘                     ├─► step-6
//           └─► step-4 ───────────┘

type JsonObject = Record<string, unknown>;
interface WorkflowOutputSentinel {}

interface WorkflowStub<TInput> {
  name: string;
  version?: string;
  defaultInput?: TInput;
}

interface TaskRef {}
interface BuiltWorkflow<TInput extends JsonObject, TOutput> {
  readonly _input?: TInput;
  readonly _output?: TOutput;
  task(opts: { name: string; parents?: TaskRef[] }): TaskRef;
}

declare function getClient(): Promise<{
  workflow<I extends JsonObject, O>(opts: { name: string; version?: string }): BuiltWorkflow<I, O>;
} | null>;

/**
 * Reusable factory that builds the workflow and its task DAG, then returns it.
 * @hatchet-workflow
 */
export async function createWorkflowBuilder<TInput extends JsonObject>(args: {
  stub: WorkflowStub<TInput>;
  langsmithApiKey?: string;
}) {
  const { stub } = args;
  const hatchet = await getClient();
  if (!hatchet) {
    throw new Error('Unable to get hatchet client');
  }

  const workflow = hatchet.workflow<TInput & JsonObject, WorkflowOutputSentinel>({
    name: stub.name,
    ...(stub.version ? { version: stub.version } : {}),
  });

  const step1 = workflow.task({ name: 'step-1' });
  const step2 = workflow.task({ name: 'step-2' });

  const step3 = workflow.task({ name: 'step-3', parents: [step1, step2] });
  const step4 = workflow.task({ name: 'step-4', parents: [step1, step2] });

  const step5 = workflow.task({ name: 'step-5', parents: [step3] });

  workflow.task({ name: 'step-6', parents: [step3, step4, step5] });

  return workflow;
}

// Build the workflow for a stub.
export const builtWorkflow = createWorkflowBuilder({ stub: { name: 'example-workflow' } });
