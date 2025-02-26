import Hatchet from '../sdk';
import { StickyStrategy, Workflow } from '../workflow';

const hatchet = Hatchet.init();

const workflow: Workflow = {
  id: 'sticky-workflow',
  description: 'test',

  steps: [
    {
      name: 'step1',
      run: async (ctx) => {
        const results: Promise<any>[] = [];
        const count = 57;
        hardChildWorkerId = undefined; // we reset this - if we run this multiple times at the same time it will break
        // eslint-disable-next-line no-plusplus
        for (let i = 0; i < count; i++) {
          const result = await ctx.spawnWorkflow(childWorkflow, {}, { sticky: true });
          results.push(result.result());
          const result2 = await ctx.spawnWorkflow(softChildWorkflow, {}, { sticky: true });
          results.push(result2.result());
        }
        console.log('Spawned ', count, ' child workflows of each type');
        console.log('Results:', await Promise.all(results));

        return { step1: 'step1 results!' };
      },
    },
  ],
};
let hardChildWorkerId: string | undefined;
const childWorkflow: Workflow = {
  id: 'child-sticky-workflow',
  description: 'test',
  sticky: StickyStrategy.HARD,
  steps: [
    {
      name: 'child-step1',
      run: async (ctx) => {
        const workerId = ctx.worker.id();

        console.log(`1: Worker ID: ${workerId}`);

        if (!hardChildWorkerId) {
          hardChildWorkerId = workerId;
        } else if (hardChildWorkerId !== workerId) {
          throw new Error(`Expected worker ID ${hardChildWorkerId} but got ${workerId}`);
        }
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

const softChildWorkflow: Workflow = {
  id: 'child-sticky-workflow-soft',
  description: 'test',
  sticky: StickyStrategy.SOFT,
  steps: [
    {
      name: 'child-step1',
      run: async (ctx) => {
        const workerId = ctx.worker.id();

        console.log(`1: Worker ID: ${workerId}`);
        return { childStep1: `SOFT ${workerId}` };
      },
    },
    {
      name: 'child-step2',
      run: async (ctx) => {
        const workerId = ctx.worker.id();
        console.log(`2: Worker ID: ${workerId}`);
        return { childStep2: `SOFT ${workerId}` };
      },
    },
  ],
};

async function main() {
  const worker1 = await hatchet.worker('sticky-worker-1');
  await worker1.registerWorkflow(workflow);
  await worker1.registerWorkflow(childWorkflow);
  await worker1.registerWorkflow(softChildWorkflow);
  worker1.start();

  const worker2 = await hatchet.worker('sticky-worker-2');
  await worker2.registerWorkflow(workflow);
  await worker2.registerWorkflow(childWorkflow);
  await worker2.registerWorkflow(softChildWorkflow);

  worker2.start();
}

main();
