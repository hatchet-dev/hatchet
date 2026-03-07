/* eslint-disable no-console */
import { Or, SleepCondition, UserEventCondition } from '@hatchet/v1/conditions';
import { NonDeterminismError } from '@hatchet/util/errors/non-determinism-error';
import sleep from '@hatchet/util/sleep';
import { hatchet } from '../hatchet-client';

export const EVENT_KEY = 'durable-example:event';
export const SLEEP_TIME_SECONDS = 2;
export const SLEEP_TIME = `${SLEEP_TIME_SECONDS}s` as const;

// > Create a durable workflow
export const durableWorkflow = hatchet.workflow({
  name: 'durable-workflow',
});
// !!

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

function extractKeyAndEventId(waitResult: unknown): { key: string; eventId: string } {
  // Eviction path returns `{ "CREATE": { [readableDataKey]: [...] } }`.
  // Pre-eviction path returns `{ [readableDataKey]: [...] }` (already unwrapped).
  const obj = waitResult as any;
  if (obj && typeof obj === 'object') {
    const key = Object.keys(obj)[0];
    const inner = obj[key];
    if (inner && typeof inner === 'object' && !Array.isArray(inner)) {
      const eventId = Object.keys(inner)[0];
      if (eventId) {
        return { key, eventId };
      }
    }
    if (key) {
      return { key: 'CREATE', eventId: key };
    }
  }

  return { key: 'CREATE', eventId: '' };
}

durableWorkflow.durableTask({
  name: 'wait_for_or_group_1',
  executionTimeout: '10m',
  fn: async (_input, ctx) => {
    const start = Date.now();
    const waitResult = await ctx.waitFor(
      Or(new SleepCondition(SLEEP_TIME, 'sleep'), new UserEventCondition(EVENT_KEY, '', 'event'))
    );
    const { key, eventId } = extractKeyAndEventId(waitResult);
    return {
      runtime: Math.round((Date.now() - start) / 1000),
      key,
      eventId,
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
    const { key, eventId } = extractKeyAndEventId(waitResult);
    return {
      runtime: Math.round((Date.now() - start) / 1000),
      key,
      eventId,
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
      return { runtime: -1 };
    }
  },
});

// --- Spawn child from durable task ---

export const spawnChildTask = hatchet.task({
  name: 'spawn-child-task',
  fn: async (input: { n?: number }) => {
    return { message: `hello from child ${input.n ?? 1}` };
  },
});

export const durableWithSpawn = hatchet.durableTask({
  name: 'durable-with-spawn',
  executionTimeout: '10s',
  fn: async (_input, ctx) => {
    const childResult = await spawnChildTask.run({});
    return { child_output: childResult };
  },
});

export const durableWithBulkSpawn = hatchet.durableTask({
  name: 'durable-with-bulk-spawn',
  executionTimeout: '10m',
  fn: async (input: { n?: number }, ctx) => {
    const n = input.n ?? 10;
    const inputs = Array.from({ length: n }, (_, i) => ({ n: i }));
    const childResults = await spawnChildTask.run(inputs);
    return { child_outputs: childResults };
  },
});

export const durableSleepEventSpawn = hatchet.durableTask({
  name: 'durable-sleep-event-spawn',
  executionTimeout: '10m',
  fn: async (_input, ctx) => {
    const start = Date.now();

    await ctx.sleepFor(SLEEP_TIME);

    await ctx.waitFor({ eventKey: EVENT_KEY });

    const childResult = await spawnChildTask.run({});

    return {
      runtime: Math.round((Date.now() - start) / 1000),
      child_output: childResult,
    };
  },
});

// --- Spawn child using explicit ctx.spawnChild ---

export const durableWithExplicitSpawn = hatchet.durableTask({
  name: 'durable-with-explicit-spawn',
  executionTimeout: '10m',
  fn: async (_input, ctx) => {
    const childResult = await ctx.spawnChild(spawnChildTask, {});
    return { child_output: childResult };
  },
});

// --- Non-determinism detection ---

export const durableNonDeterminism = hatchet.durableTask({
  name: 'durable-non-determinism',
  executionTimeout: '10s',
  fn: async (_input, ctx) => {
    const sleepTime = ctx.invocationCount * 2;

    try {
      await ctx.sleepFor(`${sleepTime}s`);
    } catch (e) {
      if (e instanceof NonDeterminismError) {
        return {
          attempt_number: ctx.invocationCount,
          sleep_time: sleepTime,
          non_determinism_detected: true,
          node_id: e.nodeId,
        };
      }
      throw e;
    }

    return {
      attempt_number: ctx.invocationCount,
      sleep_time: sleepTime,
      non_determinism_detected: false,
    };
  },
});

// --- Memo task ---

const MEMO_SLEEP_MS = 2000;

export const memoTask = hatchet.durableTask({
  name: 'memo-task',
  executionTimeout: '10m',
  fn: async (input: { message: string }, ctx) => {
    const start = Date.now();
    const res = await ctx.memo(async () => {
      await sleep(MEMO_SLEEP_MS);
      return { message: input.message, duration: MEMO_SLEEP_MS / 1000 };
    }, [input.message]);
    return { message: res.message, duration: (Date.now() - start) / 1000 };
  },
});

// --- Replay reset ---

export const REPLAY_RESET_SLEEP_SECONDS = 3;
/** Max duration (seconds) for a replayed/memoized step; above this we treat it as a real sleep. */
export const REPLAY_RESET_MEMOIZED_MAX_SECONDS = 5;
const REPLAY_RESET_SLEEP = `${REPLAY_RESET_SLEEP_SECONDS}s` as const;

export const durableReplayReset = hatchet.durableTask({
  name: 'durable-replay-reset',
  executionTimeout: '20s',
  fn: async (_input, ctx) => {
    let start = Date.now();
    await ctx.sleepFor(REPLAY_RESET_SLEEP);
    const sleep1Duration = (Date.now() - start) / 1000;

    start = Date.now();
    await ctx.sleepFor(REPLAY_RESET_SLEEP);
    const sleep2Duration = (Date.now() - start) / 1000;

    start = Date.now();
    await ctx.sleepFor(REPLAY_RESET_SLEEP);
    const sleep3Duration = (Date.now() - start) / 1000;

    return {
      sleep_1_duration: sleep1Duration,
      sleep_2_duration: sleep2Duration,
      sleep_3_duration: sleep3Duration,
    };
  },
});

// --- Spawn DAG from durable task ---

export const dagChildWorkflow = hatchet.workflow({
  name: 'dag-child-workflow-ts',
});

const dagChild1 = dagChildWorkflow.task({
  name: 'dag-child-1',
  fn: async () => {
    await sleep(1000);
    return { result: 'child1' };
  },
});

dagChildWorkflow.task({
  name: 'dag-child-2',
  parents: [dagChild1],
  fn: async () => {
    await sleep(2000);
    return { result: 'child2' };
  },
});

export const durableSpawnDag = hatchet.durableTask({
  name: 'durable-spawn-dag',
  executionTimeout: '10s',
  fn: async (_input, ctx) => {
    const sleepStart = Date.now();
    const sleepResult = await ctx.sleepFor(SLEEP_TIME);
    const sleepDuration = (Date.now() - sleepStart) / 1000;

    const spawnStart = Date.now();
    const spawnResult = await dagChildWorkflow.run({});
    const spawnDuration = (Date.now() - spawnStart) / 1000;

    return {
      sleep_duration: sleepDuration,
      sleep_result: sleepResult,
      spawn_duration: spawnDuration,
      spawn_result: spawnResult,
    };
  },
});
