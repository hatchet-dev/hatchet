import Hatchet from '../sdk';
import { Workflow } from '../workflow';

const hatchet = Hatchet.init();

// ❓ OnFailure Step
// This workflow will fail because the step will throw an error
// we define an onFailure step to handle this case

const workflow: Workflow = {
  // ... normal workflow definition
  id: 'on-failure-example',
  description: 'test',
  on: {
    event: 'user:create',
  },
  // ,
  steps: [
    {
      name: 'step1',
      run: async (ctx) => {
        // 👀 this step will always throw an error
        throw new Error('Step 1 failed');
      },
    },
  ],
  // 👀 After the workflow fails, this special step will run
  onFailure: {
    name: 'on-failure-step',
    run: async (ctx) => {
      // 👀 we can do things like perform cleanup logic
      // or notify a user here

      // 👀 you can access the error from the failed step(s) like this
      console.log(ctx.errors());

      return { onFailure: 'step' };
    },
  },
};
// ‼️

// ❓ OnFailure With Details
// Coming soon to TypeScript! https://github.com/hatchet-dev/hatchet-typescript/issues/447
// ‼️

async function main() {
  const worker = await hatchet.worker('example-worker', 1);
  await worker.registerWorkflow(workflow);
  worker.start();
}

main();
