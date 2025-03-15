import { Workflow } from '@hatchet/workflow';

export const simple: Workflow = {
  id: 'legacy-workflow',
  description: 'test',
  on: {
    event: 'user:create',
  },
  steps: [
    {
      name: 'step1',
      run: async (ctx) => {
        const input = ctx.workflowInput();

        return { step1: `original input: ${input.Message}` };
      },
    },
    {
      name: 'step2',
      parents: ['step1'],
      run: (ctx) => {
        const step1Output = ctx.stepOutput('step1');

        return { step2: `step1 output: ${step1Output.step1}` };
      },
    },
  ],
};
