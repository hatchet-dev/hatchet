import Hatchet from '@hatchet/sdk';
import { Workflow } from '@hatchet/workflow';

const hatchet = Hatchet.init();

const sleep = (ms: number) =>
  new Promise((resolve) => {
    setTimeout(resolve, ms);
  });

const workflow: Workflow = {
  id: 'example',
  description: 'test',
  on: {
    event: 'user:create',
  },
  steps: [
    {
      name: 'step1',
      run: async (input, ctx) => {
        console.log('starting step1!');
        await sleep(5000);
        console.log('executed step1!');
        return { step1: 'step1' };
      },
    },
    {
      name: 'step2',
      parents: ['step1'],
      run: (input, ctx) => {
        console.log('executed step2!', ctx.workflowInput());
        return { step2: 'step2' };
      },
    },
  ],
};

hatchet.run(workflow);
