import { hatchet } from '../hatchet-client';

export const affinityExampleTask = hatchet.task({
  name: 'affinity-example-task',
  fn: async (input, ctx) => ({ worker_id: ctx.worker.id() }),
});
