/**
 * Standard Hatchet workflow — detected automatically by the VS Code extension.
 *
 * The extension looks for `hatchet.workflow(...)` calls and renders a DAG
 * for each one.  No annotation is needed for this pattern.
 */
import Hatchet from '@hatchet-dev/typescript-sdk';

const hatchet = Hatchet.init();

// The extension detects this variable as a workflow and places a CodeLens above it.
const dataProcessing = hatchet.workflow({
  name: 'data-processing',
});

const ingest = dataProcessing.task({
  name: 'ingest',
  fn: async (input) => {
    return { raw: 'data' };
  },
});

const validate = dataProcessing.task({
  name: 'validate',
  parents: [ingest],
  fn: async (input, ctx) => {
    const { raw } = await ctx.parentOutput(ingest);
    return { valid: true };
  },
});

const transform = dataProcessing.task({
  name: 'transform',
  parents: [validate],
  fn: async (input, ctx) => {
    return { transformed: true };
  },
});

const load = dataProcessing.task({
  name: 'load',
  parents: [transform],
  fn: async (input, ctx) => {
    return { loaded: true };
  },
});
