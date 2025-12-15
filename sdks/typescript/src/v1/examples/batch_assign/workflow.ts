import { hatchet } from '../hatchet-client';

// > Declaring Types
type DagInput = {
  Message: string;
  batchId: string;
};

type DagOutput = {
  'step-1': {
    TransformedMessage: string;
  };
  'step-2': {
    Original: string;
    Transformed: string;
  };
};
// !!

// > Declaring a DAG Workflow
// First, we declare the workflow
export const dag = hatchet.workflow<DagInput, DagOutput>({
  name: 'batched-dag',
});
// !!

// > First task
// Next, we declare the tasks bound to the workflow
const toLower = dag.task({
  name: 'step-1',
  fn: (input) => {
    return {
      TransformedMessage: input.Message.toLowerCase(),
    };
  },
});
// !!

// > Second task with parent
dag.batchTask({
  name: 'step-2',
  parents: [toLower],
  batchSize: 200,
  flushInterval: 10000,
  batchKey: 'input.batchId',
  maxRuns: 1,
  fn: async (inputs, ctxs) => {
    const lowers = await Promise.all(ctxs.map((ctx) => ctx.parentOutput(toLower)));

    return lowers.map((lower, index) => ({
      Original: inputs[index].Message,
      Transformed: lower.TransformedMessage.split('').reverse().join(''),
    }));
  },
});
// !!
