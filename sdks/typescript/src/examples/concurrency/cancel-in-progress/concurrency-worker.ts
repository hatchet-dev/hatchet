import Hatchet from '../../../sdk';
import { Workflow } from '../../../workflow';

const hatchet = Hatchet.init();

const sleep = (ms: number) =>
  new Promise((resolve) => {
    setTimeout(resolve, ms);
  });

const workflow: Workflow = {
  id: 'concurrency-example',
  description: 'test',
  on: {
    event: 'concurrency:create',
  },
  concurrency: {
    name: 'user-concurrency',
    key: (ctx) => ctx.workflowInput().userId,
  },
  steps: [
    {
      name: 'step1',
      run: async (ctx) => {
        const { data } = ctx.workflowInput();
        const { signal } = ctx.controller;

        if (signal.aborted) throw new Error('step1 was aborted');

        console.log('starting step1 and waiting 5 seconds...', data);
        await sleep(5000);

        if (signal.aborted) throw new Error('step1 was aborted');

        // NOTE: the AbortController signal can be passed to many http libraries to cancel active requests
        // fetch(url, { signal })
        // axios.get(url, { signal })

        console.log('executed step1!');
        return { step1: `step1 results for ${data}!` };
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
  await worker.registerWorkflow(workflow);
  worker.start();
}

main();
