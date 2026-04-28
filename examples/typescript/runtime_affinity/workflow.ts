import { hatchet } from '../hatchet-client';

type AffinityOutput = {
  affinity_t1: { worker_id: string | undefined };
  affinity_t2: { worker_id: string | undefined };
};

export const affinityExampleTask = hatchet.workflow<{}, AffinityOutput>({
  name: 'runtime_affinity_workflow',
});

const affinityT1 = affinityExampleTask.task({
  name: 'affinity_t1',
  fn: async (input, ctx) => {
    return { worker_id: ctx.worker.id() };
  },
});

affinityExampleTask.task({
  name: 'affinity_t2',
  parents: [affinityT1],
  fn: async (input, ctx) => {
    return { worker_id: ctx.worker.id() };
  },
});
