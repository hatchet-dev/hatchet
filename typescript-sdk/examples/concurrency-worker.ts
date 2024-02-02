import Hatchet from '../src/sdk';
import { Workflow } from '../src/workflow';

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
  concurrency: {
    key: (ctx) => ctx.workflowInput().userId,
  },
  steps: [
    {
      name: 'step1',
      run: async (ctx) => {
        console.log('starting step1 and waiting 5 seconds...');
        await sleep(5000);
        console.log('executed step1!');
        return { step1: 'step1 results!' };
      },
    },
    {
      name: 'step2',
      parents: ['step1'],
      run: (ctx) => {
        console.log('executed step2 after step1 returned ', ctx.stepOutput('step1'));
        return { step2: 'step2 results!' };
      },
    },
  ],
};

hatchet.run(workflow);
