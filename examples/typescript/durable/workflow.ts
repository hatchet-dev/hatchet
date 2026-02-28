import { hatchet } from '../hatchet-client';
import { Or, SleepCondition, UserEventCondition } from '@hatchet-dev/typescript-sdk/v1/conditions';

export const EVENT_KEY = 'durable-example:event';
export const SLEEP_TIME_SECONDS = 5;
export const SLEEP_TIME = `${SLEEP_TIME_SECONDS}s` as const;

// > Create a durable workflow
export const durableWorkflow = hatchet.workflow({
  name: 'DurableWorkflow',
});

durableWorkflow.task({
  name: 'ephemeral_task',
  fn: async () => {
    console.log('Running non-durable task');
  },
});

durableWorkflow.durableTask({
  name: 'durable_task',
  executionTimeout: '10m',
  fn: async (_input, ctx) => {
    console.log('Waiting for sleep');
    await ctx.sleepFor(SLEEP_TIME);
    console.log('Sleep finished');

    console.log('Waiting for event');
    await ctx.waitFor({ eventKey: EVENT_KEY });
    console.log('Event received');

    return { status: 'success' };
  },
});

function extractKeyAndEventId(waitResult: unknown): { key: string; event_id: string } {
  // DurableContext.waitFor currently returns the CREATE payload directly.
  // The shape is typically `{ [readableDataKey]: { [eventId]: ... } }`.
  const obj = waitResult as any;
  if (obj && typeof obj === 'object') {
    const key = Object.keys(obj)[0];
    const inner = obj[key];
    if (inner && typeof inner === 'object') {
      const eventId = Object.keys(inner)[0];
      if (eventId) {
        return { key: 'CREATE', event_id: eventId };
      }
    }
    if (key) {
      return { key: 'CREATE', event_id: key };
    }
  }

  return { key: 'CREATE', event_id: '' };
}

durableWorkflow.durableTask({
  name: 'wait_for_or_group_1',
  executionTimeout: '10m',
  fn: async (_input, ctx) => {
    const start = Date.now();
    const waitResult = await ctx.waitFor(
      Or(new SleepCondition(SLEEP_TIME, 'sleep'), new UserEventCondition(EVENT_KEY, '', 'event'))
    );
    const { key, event_id } = extractKeyAndEventId(waitResult);
    return {
      runtime: Math.round((Date.now() - start) / 1000),
      key,
      event_id,
    };
  },
});

durableWorkflow.durableTask({
  name: 'wait_for_or_group_2',
  executionTimeout: '10m',
  fn: async (_input, ctx) => {
    const start = Date.now();
    const waitResult = await ctx.waitFor(
      Or(
        new SleepCondition(`${6 * SLEEP_TIME_SECONDS}s`, 'sleep'),
        new UserEventCondition(EVENT_KEY, '', 'event')
      )
    );
    const { key, event_id } = extractKeyAndEventId(waitResult);
    return {
      runtime: Math.round((Date.now() - start) / 1000),
      key,
      event_id,
    };
  },
});

durableWorkflow.durableTask({
  name: 'wait_for_multi_sleep',
  executionTimeout: '10m',
  fn: async (_input, ctx) => {
    const start = Date.now();
    // sleep 3 times
    for (let i = 0; i < 3; i += 1) {
      await ctx.sleepFor(SLEEP_TIME);
    }

    return { runtime: Math.round((Date.now() - start) / 1000) };
  },
});

export const waitForSleepTwice = hatchet.durableTask({
  name: 'wait-for-sleep-twice',
  executionTimeout: '10m',
  fn: async (_input, ctx) => {
    try {
      const start = Date.now();
      await ctx.sleepFor(SLEEP_TIME);
      return { runtime: Math.round((Date.now() - start) / 1000) };
    } catch (e) {
      // treat cancellation as a successful completion for parity with Python sample
      return { runtime: -1 };
    }
  },
});

