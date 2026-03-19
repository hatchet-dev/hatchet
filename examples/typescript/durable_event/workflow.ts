import { hatchet } from '../hatchet-client';

export const EVENT_KEY = 'user:update';

// > Durable Event
export const durableEvent = hatchet.durableTask({
  name: 'durable-event',
  executionTimeout: '10m',
  fn: async (_, ctx) => {
    const res = await ctx.waitForEvent(EVENT_KEY);

    console.log('res', res);

    return {
      Value: 'done',
    };
  },
});

export const durableEventWithFilter = hatchet.durableTask({
  name: 'durable-event-with-filter',
  executionTimeout: '10m',
  fn: async (_, ctx) => {
    // > Durable Event With Filter
    const res = await ctx.waitForEvent(EVENT_KEY, "input.userId == '1234'");

    console.log('res', res);

    return {
      Value: 'done',
    };
  },
});
