import { WorkerLabelComparator } from '@hatchet/v1';
import { hatchet } from '../hatchet-client';

// > AffinityWorkflow

const workflow = hatchet.workflow({
  name: 'affinity-workflow',
  description: 'test',
});

workflow.task({
  name: 'step1',
  fn: async (_, ctx) => {
    const results = [];
    // eslint-disable-next-line no-plusplus
    for (let i = 0; i < 50; i++) {
      const result = await childWorkflow.run({});
      results.push(result);
    }
    console.log('Spawned 50 child workflows');
    console.log('Results:', await Promise.all(results));

    return { step1: 'step1 results!' };
  },
});

// !!

// > Task with labels
const childWorkflow = hatchet.workflow({
  name: 'child-affinity-workflow',
  description: 'test',
});

childWorkflow.task({
  name: 'child-step1',
  desiredWorkerLabels: {
    model: {
      value: 'xyz',
      required: true,
    },
  },
  fn: async (ctx) => {
    return { childStep1: 'childStep1 results!' };
  },
});
// !!

childWorkflow.task({
  name: 'child-step2',
  desiredWorkerLabels: {
    memory: {
      value: 512,
      required: true,
      comparator: WorkerLabelComparator.LESS_THAN,
    },
  },
  fn: async (ctx) => {
    return { childStep2: 'childStep2 results!' };
  },
});

async function main() {
  // > AffinityWorker

  const worker1 = await hatchet.worker('affinity-worker-1', {
    labels: {
      model: 'abc',
      memory: 1024,
    },
  });

  // !!

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
