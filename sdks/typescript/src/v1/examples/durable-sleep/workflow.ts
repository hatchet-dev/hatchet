// import sleep from '@hatchet/util/sleep';
import { Or } from '@hatchet/v1/conditions';
import { hatchet } from '../hatchet-client';

export const durableSleep = hatchet.workflow({
  name: 'durable-sleep',
});

durableSleep.durableTask({
  name: 'durable-sleep',
  executionTimeout: '10m',
  fn: async (_, ctx) => {
    console.log('sleeping for 1m');
    await ctx.sleepFor('1m');
    console.log('done sleeping for 1m');
    await ctx.waitFor(
      Or(
        {
          eventKey: 'user:event',
        },
        {
          sleepFor: '2s',
        }
      )
    );
    return {
      Value: 'done',
    };
  },
});
