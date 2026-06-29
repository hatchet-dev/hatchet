// The wrapper lives here, on its own. It's consumed from orders-dag.ts in the
// same folder. For TypeScript, `@hatchet-workflow` factories are resolved across
// the whole workspace, so the usage file doesn't need to import or redeclare it
// for the DAG to render.

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
 * Reusable workflow wrapper.
 * @hatchet-workflow
 */
export function createWorkflow<TInput extends JsonObject>(stub: WorkflowStub<TInput>) {
  return getClient().workflow<TInput & JsonObject, WorkflowOutputSentinel>({
    name: stub.name,
    ...(stub.version ? { version: stub.version } : {}),
  });
}
