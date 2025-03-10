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
        return { step1: 'step1 results!' };
      },
    },
    {
      name: 'step2',
      parents: ['step1'],
      run: (ctx) => {
        return { step2: 'step2 results!' };
      },
    },
  ],
};
