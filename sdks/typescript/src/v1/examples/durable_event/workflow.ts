// import sleep from '@hatchet/util/sleep';
import { hatchet } from '../hatchet-client';

// > Durable Event
export const durableEvent = hatchet.durableTask({
  name: 'durable-event',
  executionTimeout: '10m',
  fn: async (_, ctx) => {
    const res = ctx.waitFor({
      eventKey: 'user:update',
    });

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
    const res = ctx.waitFor({
      eventKey: 'user:update',
      expression: "input.userId == '1234'",
    });
    // !!

    console.log('res', res);

    return {
      Value: 'done',
    };
  },
});
// !!
