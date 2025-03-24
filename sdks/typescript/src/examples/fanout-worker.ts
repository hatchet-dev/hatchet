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
  id: 'parent-workflow',
  description: 'simple example for spawning child workflows',
  on: {
    event: 'fanout:create',
  },
  steps: [
    {
      name: 'parent-spawn',
      timeout: '70s',
      run: async (ctx) => {
        const promises = Array.from({ length: 3 }, (_, i) =>
          ctx
            .spawnWorkflow<
              Input,
              Output
            >('child-workflow', { input: `child-input-${i}` }, { additionalMetadata: { childKey: 'childValue' } })
            .output.then((result) => {
              ctx.log('spawned workflow result:');
              return result;
            })
        );

        const results = await Promise.all(promises);
        console.log('spawned workflow results:', results);
        console.log('number of spawned workflows:', results.length);
        return { spawned: 'x' };
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

        const garbageData = Array.from({ length: 1e6 / 3.5 }, (_, i) => `garbage-${i}`).join(',');
        console.log('Generated garbage data:', `${garbageData.slice(0, 100)}...`); // Print a snippet of the garbage data
        return { 'child-output': garbageData };
      },
    },
    {
      name: 'child-work3',
      parents: ['child-work'],
      run: async (ctx) => {
        const { input } = ctx.workflowInput();
        // throw new Error('child error');
        const garbageData = Array.from({ length: 1e6 / 3.5 }, (_, i) => `garbage-${i}`).join(',');
        console.log('Generated garbage data:', `${garbageData.slice(0, 100)}...`); // Print a snippet of the garbage data
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
