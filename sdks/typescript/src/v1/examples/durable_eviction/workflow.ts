/* eslint-disable no-console */
import sleep from '@hatchet/util/sleep';
import { EvictionPolicy } from '@hatchet/v1';
import { hatchet } from '../hatchet-client';

export const EVICTION_TTL_SECONDS = 5;
export const LONG_SLEEP_SECONDS = 15;
export const EVENT_KEY = 'durable-eviction:event';

const EVICTION_POLICY: EvictionPolicy = {
  ttl: `${EVICTION_TTL_SECONDS}s`,
  allowCapacityEviction: true,
  priority: 0,
};

export const childTask = hatchet.task({
  name: 'eviction-child-task',
  fn: async () => {
    await sleep(LONG_SLEEP_SECONDS * 1000);
    return { child_status: 'completed' };
  },
});

export const evictableSleep = hatchet.durableTask({
  name: 'evictable-sleep',
  executionTimeout: '5m',
  evictionPolicy: EVICTION_POLICY,
  fn: async (_input, ctx) => {
    await ctx.sleepFor(`${LONG_SLEEP_SECONDS}s`);
    return { status: 'completed' };
  },
});

export const evictableWaitForEvent = hatchet.durableTask({
  name: 'evictable-wait-for-event',
  executionTimeout: '5m',
  evictionPolicy: EVICTION_POLICY,
  fn: async (_input, ctx) => {
    await ctx.waitFor({ eventKey: EVENT_KEY });
    return { status: 'completed' };
  },
});

export const evictableChildSpawn = hatchet.durableTask({
  name: 'evictable-child-spawn',
  executionTimeout: '5m',
  evictionPolicy: EVICTION_POLICY,
  fn: async (_input, ctx) => {
    const childResult = await childTask.run({});
    return { child: childResult, status: 'completed' };
  },
});

export const multipleEviction = hatchet.durableTask({
  name: 'multiple-eviction',
  executionTimeout: '5m',
  evictionPolicy: EVICTION_POLICY,
  fn: async (_input, ctx) => {
    await ctx.sleepFor(`${LONG_SLEEP_SECONDS}s`);
    await ctx.sleepFor(`${LONG_SLEEP_SECONDS}s`);
    return { status: 'completed' };
  },
});

export const nonEvictableSleep = hatchet.durableTask({
  name: 'non-evictable-sleep',
  executionTimeout: '5m',
  evictionPolicy: {
    ttl: undefined,
    allowCapacityEviction: false,
    priority: 0,
  },
  fn: async (_input, ctx) => {
    await ctx.sleepFor('10s');
    return { status: 'completed' };
  },
});
