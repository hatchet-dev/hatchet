import { hatchet } from '../hatchet-client';

export const EVENT_KEY = 'user:update';
export const SCOPE = 'user-1234';

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
// !!

export const durableEventWithFilter = hatchet.durableTask({
  name: 'durable-event-with-filter',
  executionTimeout: '10m',
  fn: async (_, ctx) => {
    // > Durable Event With Filter
    const res = await ctx.waitForEvent(EVENT_KEY, "input.userId == '1234'");
    // !!

    console.log('res', res);

    return {
      Value: 'done',
    };
  },
});
// !!

// > Durable Event With Lookback
export const durableEventWithLookback = hatchet.durableTask({
  name: 'durable-event-with-lookback',
  executionTimeout: '10m',
  fn: async (_, ctx) => {
    const res = await ctx.waitForEvent(EVENT_KEY, undefined, undefined, SCOPE, '1m');

    console.log('res', res);

    return {
      Value: 'done',
    };
  },
});
// !!
