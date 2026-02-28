import Hatchet from '../sdk';
import { Workflow } from '../workflow';

const hatchet = Hatchet.init();

const sleep = (ms: number) =>
  new Promise((resolve) => {
    setTimeout(resolve, ms);
  });

export const simpleWorkflow: Workflow = {
  id: 'simple-workflow',
  description: 'test',
  on: {
    event: 'user:create',
  },
  steps: [
    {
      name: 'step1',
      run: async (ctx) => {
        console.log('starting step1 with the following input', ctx.workflowInput());
        console.log('waiting 5 seconds...');
        await sleep(5000);
        ctx.putStream('step1 stream');
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

async function main() {
  const worker = await hatchet.worker('example-worker');
  await worker.registerWorkflow(simpleWorkflow);
  worker.start();
}

if (require.main === module) {
  main();
}
