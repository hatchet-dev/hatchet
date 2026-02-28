import Hatchet from '../sdk';
import { StickyStrategy, Workflow } from '../workflow';

const hatchet = Hatchet.init();

// > StickyWorker

const workflow: Workflow = {
  id: 'sticky-workflow',
  description: 'test',
  steps: [
    {
      name: 'step1',
      run: async (ctx) => {
        const results: Promise<any>[] = [];

        // eslint-disable-next-line no-plusplus
        for (let i = 0; i < 50; i++) {
          const result = await ctx.spawnWorkflow(childWorkflow, {}, { sticky: true });
          results.push(result.result());
        }
        console.log('Spawned 50 child workflows');
        console.log('Results:', await Promise.all(results));

        return { step1: 'step1 results!' };
      },
    },
  ],
};

// !!

// > StickyChild

const childWorkflow: Workflow = {
  id: 'child-sticky-workflow',
  description: 'test',
  // 👀 Specify a sticky strategy when declaring the workflow
  sticky: StickyStrategy.HARD,
  steps: [
    {
      name: 'child-step1',
      run: async (ctx) => {
        const workerId = ctx.worker.id();

        console.log(`1: Worker ID: ${workerId}`);
        return { childStep1: `${workerId}` };
      },
    },
    {
      name: 'child-step2',
      run: async (ctx) => {
        const workerId = ctx.worker.id();
        console.log(`2: Worker ID: ${workerId}`);
        return { childStep2: `${workerId}` };
      },
    },
  ],
};

// !!

async function main() {
  const worker1 = await hatchet.worker('sticky-worker-1');
  await worker1.registerWorkflow(workflow);
  await worker1.registerWorkflow(childWorkflow);
  worker1.start();

  const worker2 = await hatchet.worker('sticky-worker-2');
  await worker2.registerWorkflow(workflow);
  await worker2.registerWorkflow(childWorkflow);
  worker2.start();
}

main();
