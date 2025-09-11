import Hatchet from '../../../sdk';
import { ConcurrencyLimitStrategy, Workflow } from '../../../workflow';

const hatchet = Hatchet.init();

const sleep = (ms: number) =>
  new Promise((resolve) => {
    setTimeout(resolve, ms);
  });

const workflow: Workflow = {
  id: 'concurrency-example-rr',
  description: 'test',
  on: {
    event: 'concurrency:create',
  },
  concurrency: {
    name: 'user-concurrency',
    expression: 'input.group',
    maxRuns: 2,
    limitStrategy: ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
  },
  steps: [
    {
      name: 'step1',
      run: async (ctx) => {
        const { data } = ctx.workflowInput();
        const { signal } = ctx.controller;

        if (signal.aborted) throw new Error('step1 was aborted');

        console.log('starting step1 and waiting 5 seconds...', data);
        await sleep(2000);

        if (signal.aborted) throw new Error('step1 was aborted');

        // NOTE: the AbortController signal can be passed to many http libraries to cancel active requests
        // fetch(url, { signal })
        // axios.get(url, { signal })

        console.log('executed step1!');
        return { step1: `step1 results for ${data}!` };
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
