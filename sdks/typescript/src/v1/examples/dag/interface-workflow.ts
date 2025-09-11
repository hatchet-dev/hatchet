import { WorkflowInputType, WorkflowOutputType } from '@hatchet/v1';
import { hatchet } from '../hatchet-client';

interface DagInput extends WorkflowInputType {
  Message: string;
}

interface DagOutput extends WorkflowOutputType {
  reverse: {
    Original: string;
    Transformed: string;
  };
}

// > Declaring a DAG Workflow
// First, we declare the workflow
export const dag = hatchet.workflow<DagInput, DagOutput>({
  name: 'simple',
});

const reverse = dag.task({
  name: 'reverse',
  fn: (input) => {
    return {
      Original: input.Message,
      Transformed: input.Message.split('').reverse().join(''),
    };
  },
});

dag.task({
  name: 'to-lower',
  parents: [reverse],
  fn: async (input, ctx) => {
    const r = await ctx.parentOutput(reverse);

    return {
      reverse: {
        Original: r.Transformed,
        Transformed: r.Transformed.toLowerCase(),
      },
    };
  },
});
