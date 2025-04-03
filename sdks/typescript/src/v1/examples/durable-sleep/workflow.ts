// import sleep from '@hatchet/util/sleep';
import { Or } from '@hatchet/v1/conditions';
import { hatchet } from '../hatchet-client';

export const durableSleep = hatchet.workflow({
  name: 'durable-sleep',
});

// ❓ Durable Sleep
durableSleep.durableTask({
  name: 'durable-sleep',
  executionTimeout: '10m',
  fn: async (_, ctx) => {
    console.log('sleeping for 5s');
    const sleepRes = await ctx.sleepFor('5s');
    console.log('done sleeping for 5s', sleepRes);

    return {
      Value: 'done',
    };
  },
});
// !!
