import Hatchet from '../sdk';
import { Workflow } from '../workflow';

const hatchet = Hatchet.init({
  log_level: 'OFF',
});

const sleep = (ms: number) =>
  new Promise((resolve) => {
    setTimeout(resolve, ms);
  });

const workflow: Workflow = {
  id: 'dag-example',
  description: 'test',
  on: {
    event: 'user:create',
  },
  steps: [
    {
      name: 'dag-step1',
      run: async (ctx) => {
        console.log('executed step1!');
        await sleep(5000);
        return { step1: 'step1' };
      },
    },
    {
      name: 'dag-step2',
      parents: ['dag-step1'],
      run: async (ctx) => {
        console.log('executed step2!');
        await sleep(5000);
        return { step2: 'step2' };
      },
    },
    {
      name: 'dag-step3',
      parents: ['dag-step1', 'dag-step2'],
      run: (ctx) => {
        console.log('executed step3!');
        return { step3: 'step3' };
      },
    },
    {
      name: 'dag-step4',
      parents: ['dag-step1', 'dag-step3'],
      run: async (ctx) => {
        await sleep(5000);

        // simulate a really slow network call
        setTimeout(async () => {
          await sleep(1000);
          ctx.playground('slow', 'call');
        }, 5000);

        return { step4: 'step4' };
      },
    },
  ],
};

async function main() {
  const worker = await hatchet.worker('example-worker', 1);
  await worker.registerWorkflow(workflow);
  worker.start();
}

main();
