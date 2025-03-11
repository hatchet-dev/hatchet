import { hatchet } from '../client';

type SimpleWorkflowInput = {
  Message: string;
};

type SimpleWorkflowOutput = {
  step2: {
    Output: string;
  };
};

export const simple = hatchet.createWorkflow<SimpleWorkflowInput, SimpleWorkflowOutput>({
  name: 'simple',
});

const step1 = simple.addTask({
  name: 'step1',
  fn: () => {
    return {
      Result: true,
    };
  },
});

simple.addTask({
  name: 'step2',
  parents: [step1],
  fn: async (_, ctx) => {
    const parent1Result = ctx.parentData(step1);

    if (parent1Result.Result) {
      return {
        Output: 'true',
      };
    }
    throw new Error('Function not implemented.');
  },
});
