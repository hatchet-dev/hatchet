import sleep from '@hatchet-dev/typescript-sdk/util/sleep';
import { EvictionPolicy } from '@hatchet-dev/typescript-sdk/v1';
import { hatchet } from '../hatchet-client';

export const EVICTION_TTL_SECONDS = 5;
export const LONG_SLEEP_SECONDS = 15;
export const EVENT_KEY = 'durable-eviction:event';

// > Eviction Policy
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

// > Evictable Sleep
export const evictableSleep = hatchet.durableTask({
  name: 'evictable-sleep',
  executionTimeout: '5m',
  evictionPolicy: EVICTION_POLICY,
  fn: async (_input, ctx) => {
    await ctx.sleepFor(`${LONG_SLEEP_SECONDS}s`);
    return { status: 'completed' };
  },
});

// NOTE: DO NOT REGISTER ON E2E TEST WORKER
export const evictableSleepForGracefulTermination = hatchet.durableTask({
  name: 'evictable-sleep-for-graceful-termination',
  executionTimeout: '5m',
  evictionPolicy: {
    ttl: `30m`,
    allowCapacityEviction: true,
    priority: 0,
  },
  fn: async (_input, ctx) => {
    await ctx.sleepFor(`5m`);
    return { status: 'completed' };
  },
});

export const evictableWaitForEvent = hatchet.durableTask({
  name: 'evictable-wait-for-event',
  executionTimeout: '5m',
  evictionPolicy: EVICTION_POLICY,
  fn: async (_input, ctx) => {
    await ctx.waitForEvent(EVENT_KEY, 'true');
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

export const bulkChildTask = hatchet.task({
  name: 'eviction-bulk-child-task',
  fn: async (input: { sleepSeconds: number }) => {
    await sleep(input.sleepSeconds * 1000);
    return { sleepSeconds: input.sleepSeconds, status: 'completed' };
  },
});

export const evictableChildBulkSpawn = hatchet.durableTask({
  name: 'evictable-child-bulk-spawn',
  executionTimeout: '5m',
  evictionPolicy: EVICTION_POLICY,
  fn: async (_input, ctx) => {
    const inputs = Array.from({ length: 3 }, (_, i) => ({
      sleepSeconds: (EVICTION_TTL_SECONDS + 5) * (i + 1),
    }));
    const childResults = await bulkChildTask.run(inputs);
    return { child_results: childResults, status: 'completed' };
  },
});

export const CAPACITY_SLEEP_SECONDS = 20;

export const capacityEvictableSleep = hatchet.durableTask({
  name: 'capacity-evictable-sleep',
  executionTimeout: '5m',
  evictionPolicy: {
    ttl: undefined,
    allowCapacityEviction: true,
    priority: 0,
  },
  fn: async (_input, ctx) => {
    await ctx.sleepFor(`${CAPACITY_SLEEP_SECONDS}s`);
    return { status: 'completed' };
  },
});

// > Non Evictable Sleep
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
