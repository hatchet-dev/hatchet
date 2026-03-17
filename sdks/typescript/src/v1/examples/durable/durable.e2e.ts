import sleep from '@hatchet/util/sleep';
import { makeE2EClient, checkDurableEvictionSupport } from '../__e2e__/harness';
import {
  durableWorkflow,
  EVENT_KEY,
  SLEEP_TIME_SECONDS,
  REPLAY_RESET_SLEEP_SECONDS,
  REPLAY_RESET_MEMOIZED_MAX_SECONDS,
  waitForSleepTwice,
  durableWithSpawn,
  durableWithBulkSpawn,
  durableSleepEventSpawn,
  durableSpawnDag,
  durableNonDeterminism,
  durableReplayReset,
} from './workflow';

describe('durable-e2e', () => {
  const hatchet = makeE2EClient();
  let evictionSupported = false;

  beforeAll(async () => {
    evictionSupported = await checkDurableEvictionSupport(hatchet);
  });

  function requireEviction() {
    if (!evictionSupported) {
      console.log('Skipping: engine does not support durable eviction');
    }
    return !evictionSupported;
  }

  it('durable workflow waits for sleep + event', async () => {
    const ref = await durableWorkflow.runNoWait({});

    let finished = false;
    const resultPromise = ref.output.finally(() => {
      finished = true;
    });

    const eventPusher = (async () => {
      await sleep((SLEEP_TIME_SECONDS + 1) * 1000);
      for (let i = 0; i < 30 && !finished; i += 1) {
        await hatchet.events.push(EVENT_KEY, { test: 'test', i });
        await sleep(200);
      }
    })();

    const result = await resultPromise;
    await eventPusher.catch(() => undefined);

    const workers = await hatchet.workers.list();
    expect(workers.rows?.length).toBeGreaterThan(0);

    const activeWorkers = (workers.rows || []).filter((w: any) => w.status === 'ACTIVE');
    expect(activeWorkers.length).toBeGreaterThanOrEqual(1);
    expect(activeWorkers.some((w: any) => `${w.name}`.includes('e2e-test-worker'))).toBeTruthy();

    expect((result as any).durable_task.status).toBe('success');

    const g1 = (result as any).wait_for_or_group_1;
    const g2 = (result as any).wait_for_or_group_2;

    expect(Math.abs(g1.runtime - SLEEP_TIME_SECONDS)).toBeLessThanOrEqual(5);
    expect(g1.key).toBe(g2.key);
    expect(g1.key).toBe('CREATE');
    expect(['sleep', 'event']).toContain(`${g1.eventId}`);
    expect(['sleep', 'event']).toContain(`${g2.eventId}`);

    const multi = (result as any).wait_for_multi_sleep;
    expect(multi.runtime).toBeGreaterThan(3 * SLEEP_TIME_SECONDS);
  }, 300_000);

  it('durable sleep cancel + replay', async () => {
    if (requireEviction()) {return;}
    const ref = await waitForSleepTwice.runNoWait({});

    await sleep((SLEEP_TIME_SECONDS * 1000) / 2);
    await ref.cancel();

    await ref.output.catch(() => undefined);

    await ref.replay();

    const replayed = await ref.output;
    expect(replayed.runtime).toBeLessThan(SLEEP_TIME_SECONDS + 3);
  }, 300_000);

  it('durable child spawn', async () => {
    const result = await durableWithSpawn.run({});
    expect(result.child_output).toEqual({ message: 'hello from child 1' });
  }, 300_000);

  it('durable child bulk spawn', async () => {
    const n = 10;
    const result = await durableWithBulkSpawn.run({ n });
    expect(result.child_outputs).toEqual(
      Array.from({ length: n }, (_, i) => ({ message: `hello from child ${i}` }))
    );
  }, 300_000);

  it('durable sleep + event + spawn replay', async () => {
    if (requireEviction()) {return;}
    const start = Date.now();
    const ref = await durableSleepEventSpawn.runNoWait({});

    let finished = false;
    const resultPromise = ref.output.finally(() => {
      finished = true;
    });

    const eventPusher = (async () => {
      await sleep((SLEEP_TIME_SECONDS + 1) * 1000);
      for (let i = 0; i < 30 && !finished; i += 1) {
        await hatchet.events.push(EVENT_KEY, { test: 'test', i });
        await sleep(200);
      }
    })();

    const result = await resultPromise;
    await eventPusher.catch(() => undefined);
    const firstElapsed = (Date.now() - start) / 1000;

    expect(result.child_output).toEqual({ message: 'hello from child 1' });
    expect(firstElapsed).toBeGreaterThanOrEqual(SLEEP_TIME_SECONDS);

    const replayStart = Date.now();
    await ref.replay();
    const replayed = await ref.output;
    const replayElapsed = (Date.now() - replayStart) / 1000;

    expect(replayed.child_output).toEqual({ message: 'hello from child 1' });
    expect(replayElapsed).toBeLessThan(SLEEP_TIME_SECONDS);
  }, 300_000);

  it('durable completed replay', async () => {
    if (requireEviction()) {return;}
    const ref = await waitForSleepTwice.runNoWait({});

    const start = Date.now();
    const firstResult = await ref.output;
    const firstElapsed = (Date.now() - start) / 1000;

    expect(firstResult.runtime).toBeGreaterThanOrEqual(SLEEP_TIME_SECONDS);
    expect(firstElapsed).toBeGreaterThanOrEqual(SLEEP_TIME_SECONDS);

    const replayStart = Date.now();
    await ref.replay();
    const replayed = await ref.output;
    const replayElapsed = (Date.now() - replayStart) / 1000;

    expect(replayed.runtime).toBeLessThan(SLEEP_TIME_SECONDS);
    expect(replayElapsed).toBeLessThan(SLEEP_TIME_SECONDS);
  }, 300_000);

  it('durable spawn DAG', async () => {
    const start = Date.now();
    const result = await durableSpawnDag.run({});
    const elapsed = (Date.now() - start) / 1000;

    expect(result.sleep_duration).toBeGreaterThanOrEqual(SLEEP_TIME_SECONDS);
    expect(result.spawn_duration).toBeGreaterThanOrEqual(1);
    expect(elapsed).toBeGreaterThanOrEqual(SLEEP_TIME_SECONDS);
    expect(elapsed).toBeLessThanOrEqual(60);
  }, 300_000);

  it('durable non-determinism', async () => {
    if (requireEviction()) {return;}
    const ref = await durableNonDeterminism.runNoWait({});
    const result = await ref.output;

    expect(result.non_determinism_detected).toBe(false);

    await ref.replay();
    const replayed = await ref.output;

    expect(replayed.non_determinism_detected).toBe(true);
    expect(replayed.node_id).toBe(1);
    expect(replayed.attempt_number).toBe(2);
  }, 300_000);

  it.each([1, 2, 3])(
    'durable replay reset from node %i',
    async (nodeId) => {
      if (requireEviction()) {return;}
      const ref = await durableReplayReset.runNoWait({});
      const result = await ref.output;

      expect(result.sleep_1_duration).toBeGreaterThanOrEqual(REPLAY_RESET_SLEEP_SECONDS);
      expect(result.sleep_2_duration).toBeGreaterThanOrEqual(REPLAY_RESET_SLEEP_SECONDS);
      expect(result.sleep_3_duration).toBeGreaterThanOrEqual(REPLAY_RESET_SLEEP_SECONDS);

      const runId = await ref.getWorkflowRunId();
      const taskExternalId = await hatchet.runs.getTaskExternalId(runId);
      await hatchet.runs.branchDurableTask(taskExternalId, nodeId);
      await sleep('1s');

      const resetStart = Date.now();
      const resetResult = await ref.output;
      const resetElapsed = (Date.now() - resetStart) / 1000;

      const durations = [
        resetResult.sleep_1_duration,
        resetResult.sleep_2_duration,
        resetResult.sleep_3_duration,
      ];

      for (let i = 0; i < durations.length; i += 1) {
        if (i + 1 >= nodeId) {
          expect(durations[i]).toBeGreaterThanOrEqual(REPLAY_RESET_SLEEP_SECONDS);
        } else {
          expect(durations[i]).toBeLessThan(REPLAY_RESET_MEMOIZED_MAX_SECONDS);
        }
      }

      const sleepsToDo = 3 - nodeId + 1;
      expect(resetElapsed).toBeGreaterThanOrEqual(sleepsToDo * REPLAY_RESET_SLEEP_SECONDS);
    },
    300_000
  );
});
