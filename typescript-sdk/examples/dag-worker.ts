import Hatchet from '../src/sdk';
import { Workflow } from '../src/workflow';

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
      run: (ctx) => {
        console.log('executed step2!');
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
      run: (ctx) => {
        console.log('executed step4!');
        return { step4: 'step4' };
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
