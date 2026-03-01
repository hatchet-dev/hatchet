import sleep from '@hatchet/util/sleep';
import { hatchet } from '../hatchet-client';

export type MockInput = { foo: string };

export const runDetailTestWorkflow = hatchet.workflow<MockInput>({
  name: 'run-detail-test',
});

const step1 = runDetailTestWorkflow.task({
  name: 'step1',
  fn: async () => ({
    random_number: Math.floor(Math.random() * 100) + 1,
  }),
});

const step2 = runDetailTestWorkflow.task({
  name: 'step2',
  fn: async () => {
    await sleep(5000);
    return {
      random_number: Math.floor(Math.random() * 100) + 1,
    };
  },
});

runDetailTestWorkflow.task({
  name: 'cancel_step',
  fn: async (_input, ctx) => {
    await ctx.cancel();
    await sleep(10000);
    return {};
  },
});

runDetailTestWorkflow.task({
  name: 'fail_step',
  fn: async () => {
    throw new Error('Intentional Failure');
  },
});

const step3 = runDetailTestWorkflow.task({
  name: 'step3',
  parents: [step1, step2],
  fn: async (_input, ctx) => {
    const one = (await ctx.parentOutput(step1)).random_number;
    const two = (await ctx.parentOutput(step2)).random_number;
    return { sum: one + two };
  },
});

runDetailTestWorkflow.task({
  name: 'step4',
  parents: [step1, step3],
  fn: async () => ({ step4: 'step4' }),
});
