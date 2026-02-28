import sleep from '@hatchet/util/sleep';
import { makeE2EClient, startWorker, stopWorker } from '../__e2e__/harness';
import { durableWorkflow, EVENT_KEY, SLEEP_TIME_SECONDS, waitForSleepTwice } from './workflow';

xdescribe('durable-e2e', () => {
  const hatchet = makeE2EClient();
  let worker: Awaited<ReturnType<typeof startWorker>> | undefined;

  beforeAll(async () => {
    worker = await startWorker({
      client: hatchet,
      name: 'e2e-test-worker',
      workflows: [durableWorkflow, waitForSleepTwice],
      slots: 10,
    });
  });

  afterAll(async () => {
    await stopWorker(worker);
  });

  it('durable workflow waits for sleep + event', async () => {
    const ref = await durableWorkflow.runNoWait({});

    // `runNoWait` returns before work starts; reliably getting an event to `durable_task`
    // means we need to push events *until* the task is actually waiting.
    let finished = false;
    const resultPromise = ref.output.finally(() => {
      finished = true;
    });

    const eventPusher = (async () => {
      // Push a handful of events over time to handle single-consumer semantics.
      // Delay pushing so `wait_for_or_group_1` resolves via its sleep condition.
      await sleep((SLEEP_TIME_SECONDS + 1) * 1000);
      for (let i = 0; i < 30 && !finished; i += 1) {
        await hatchet.events.push(EVENT_KEY, { test: 'test', i });
        await sleep(1000);
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

    // runtime is rounded to seconds and can drift a bit under load
    expect(Math.abs(g1.runtime - SLEEP_TIME_SECONDS)).toBeLessThanOrEqual(5);
    expect(g1.key).toBe(g2.key);
    expect(g1.key).toBe('CREATE');
    expect(`${g1.event_id}`).toContain('sleep');
    expect(`${g2.event_id}`).toContain('event');

    const multi = (result as any).wait_for_multi_sleep;
    expect(multi.runtime).toBeGreaterThan(3 * SLEEP_TIME_SECONDS);
  }, 180_000);

  it('durable sleep cancel + replay', async () => {
    const ref = await waitForSleepTwice.runNoWait({});

    await sleep((SLEEP_TIME_SECONDS * 1000) / 2);
    await ref.cancel();

    // may resolve or reject depending on engine; we only need it to settle
    await ref.output.catch(() => undefined);

    await ref.replay();

    const replayed = await ref.output;
    // We've already slept a bit by the time the task is cancelled
    expect(replayed.runtime).toBeLessThanOrEqual(SLEEP_TIME_SECONDS);
  }, 180_000);
});
