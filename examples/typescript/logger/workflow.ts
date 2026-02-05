import { hatchet } from '../hatchet-client';

// Mirrors `sdks/python/examples/logger/workflow.py`
export const loggingWorkflow = hatchet.workflow({
  name: 'LoggingWorkflow',
});

loggingWorkflow.task({
  name: 'root_logger',
  fn: async () => {
    for (let i = 0; i < 12; i += 1) {
      console.info(`executed step1 - ${i}`);
      console.info({ step1: 'step1' });
      // keep this fast for e2e
    }

    return { status: 'success' };
  },
});

loggingWorkflow.task({
  name: 'context_logger',
  fn: async (_input, ctx) => {
    for (let i = 0; i < 12; i += 1) {
      // Python uses ctx.log; TS has both ctx.log (deprecated) and ctx.logger.*
      // Use ctx.log to stay closer semantically.
      await ctx.log(`executed step1 - ${i}`);
      await ctx.log(JSON.stringify({ step1: 'step1' }));
    }

    return { status: 'success' };
  },
});


