import sleep from '@hatchet/util/sleep';
import { hatchet } from '../client';

export const durableSleep = hatchet.workflow({
  name: 'durable-sleep',
});

durableSleep.task({
  name: 'non-durable-sleep',
  fn: async (input, ctx) => {
    await sleep(1000);

    return {
      Value: 'done',
    };
  },
});

durableSleep.durableTask({
  name: 'durable-sleep',
  fn: async (input, ctx) => {
    await ctx.sleepFor(1000);

    return {
      Value: 'done',
    };
  },
});
