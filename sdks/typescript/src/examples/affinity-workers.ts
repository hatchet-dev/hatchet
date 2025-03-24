import { WorkerLabelComparator } from '@hatchet/protoc/workflows';
import Hatchet from '../sdk';
import { Workflow } from '../workflow';

const hatchet = Hatchet.init();

const workflow: Workflow = {
  id: 'affinity-workflow',
  description: 'test',
  steps: [
    {
      name: 'step1',
      run: async (ctx) => {
        const results: Promise<any>[] = [];
        // eslint-disable-next-line no-plusplus
        for (let i = 0; i < 50; i++) {
          const result = await ctx.spawnWorkflow(childWorkflow.id, {});
          results.push(result.output);
        }
        console.log('Spawned 50 child workflows');
        console.log('Results:', await Promise.all(results));

        return { step1: 'step1 results!' };
      },
    },
  ],
};

const childWorkflow: Workflow = {
  id: 'child-affinity-workflow',
  description: 'test',
  steps: [
    {
      name: 'child-step1',
      worker_labels: {
        model: {
          value: 'xyz',
          required: true,
        },
      },
      run: async (ctx) => {
        console.log('starting child-step1 with the following input', ctx.workflowInput());
        return { childStep1: 'childStep1 results!' };
      },
    },
    {
      name: 'child-step2',
      worker_labels: {
        memory: {
          value: 512,
          required: true,
          comparator: WorkerLabelComparator.LESS_THAN,
        },
      },
      run: async (ctx) => {
        console.log('starting child-step2 with the following input', ctx.workflowInput());
        return { childStep2: 'childStep2 results!' };
      },
    },
  ],
};

async function main() {
  const worker1 = await hatchet.worker('affinity-worker-1', {
    labels: {
      model: 'abc',
      memory: 1024,
    },
  });
  await worker1.registerWorkflow(workflow);
  await worker1.registerWorkflow(childWorkflow);
  worker1.start();

  const worker2 = await hatchet.worker('affinity-worker-2', {
    labels: {
      model: 'xyz',
      memory: 512,
    },
  });
  await worker2.registerWorkflow(workflow);
  await worker2.registerWorkflow(childWorkflow);
  worker2.start();
}

main();
