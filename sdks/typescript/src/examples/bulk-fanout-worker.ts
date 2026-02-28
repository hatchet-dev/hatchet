import Hatchet from '../sdk';
import { Workflow } from '../workflow';

const hatchet = Hatchet.init();

type Input = {
  input: string;
};

type Output = {
  'child-work': {
    'child-output': string;
  };
};

const parentWorkflow: Workflow = {
  id: 'bulk-parent-workflow',
  description: 'simple example for spawning child workflows',
  on: {
    event: 'bulk:fanout:create',
  },
  steps: [
    {
      name: 'parent-spawn',
      timeout: '70s',
      run: async (ctx) => {
        // Prepare the workflows to spawn
        const workflowRequests = Array.from({ length: 300 }, (_, i) => ({
          workflow: 'child-workflow',
          input: { input: `child-input-${i}` },
          options: { additionalMetadata: { childKey: 'childValue' } },
        }));

        const spawnedWorkflows = await ctx.spawnWorkflows<Input, Output>(workflowRequests);

        const results = await Promise.all(
          spawnedWorkflows.map((workflowRef) =>
            workflowRef.output.then((result) => {
              ctx.logger.info('spawned workflow result:');
              return result;
            })
          )
        );

        console.log('spawned workflow results:', results);
        console.log('number of spawned workflows:', results.length);
        return { spawned: results.length };
      },
    },
  ],
};

const childWorkflow: Workflow = {
  id: 'child-workflow',
  description: 'simple example for spawning child workflows',
  on: {
    event: 'child:create',
  },
  steps: [
    {
      name: 'child-work',
      run: async (ctx) => {
        const { input } = ctx.workflowInput();
        // throw new Error('child error');
        return { 'child-output': 'sm' };
      },
    },
    {
      name: 'child-work2',
      run: async (ctx) => {
        const { input } = ctx.workflowInput();
        // Perform CPU-bound work
        // throw new Error('child error');
        console.log('child workflow input:', input);
        // Generate a large amount of garbage data

        const garbageData = 'garbage'; // Print a snippet of the garbage data
        return { 'child-output': garbageData };
      },
    },
    {
      name: 'child-work3',
      parents: ['child-work'],
      run: async (ctx) => {
        const { input } = ctx.workflowInput();
        // throw new Error('child error');
        const garbageData = 'child garbage';
        return { 'child-output': garbageData };
      },
    },
  ],
};

async function main() {
  const worker = await hatchet.worker('fanout-worker', { maxRuns: 1000 });
  await worker.registerWorkflow(parentWorkflow);
  await worker.registerWorkflow(childWorkflow);
  worker.start();
}

main();
