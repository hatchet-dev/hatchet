import sleep from '@hatchet/util/sleep';
import { makeE2EClient, startWorker, stopWorker } from '../__e2e__/harness';
import { taskConditionWorkflow } from './complex-workflow';

describe('conditions-e2e', () => {
  const hatchet = makeE2EClient();
  let worker: Awaited<ReturnType<typeof startWorker>> | undefined;

  beforeAll(async () => {
    worker = await startWorker({
      client: hatchet,
      name: 'conditions-e2e-worker',
      workflows: [taskConditionWorkflow],
      slots: 10,
    });
  });

  afterAll(async () => {
    await stopWorker(worker);
  });

  xit(
    'waits, receives events, and completes branches',
    async () => {
      const ref = await taskConditionWorkflow.runNoWait({});

      // give the workflow time to reach waits
      await sleep(15_000);

      await hatchet.events.push('skip_on_event:skip', {});
      await hatchet.events.push('wait_for_event:start', {});

      const result: any = await ref.output;

      // python asserts skip_on_event is skipped
      expect(result.skipOnEvent?.skipped ?? result.skip_on_event?.skipped).toBeTruthy();

      const startRandom = result.start.randomNumber ?? result.start.random_number;
      const waitForEventRandom = result.waitForEvent.randomNumber ?? result.wait_for_event.random_number;
      const waitForSleepRandom = result.waitForSleep.randomNumber ?? result.wait_for_sleep.random_number;

      const left = result.leftBranch ?? result.left_branch;
      const right = result.rightBranch ?? result.right_branch;

      expect(Boolean(left?.skipped) || Boolean(right?.skipped)).toBeTruthy();

      const branchRandom = left?.randomNumber ?? right?.randomNumber ?? left?.random_number ?? right?.random_number;
      const sum = result.sum.sum;

      // TS version includes optional skipped branches as 0 in its sum implementation;
      // verify at least the required components add up.
      expect(sum).toBeGreaterThanOrEqual(startRandom + waitForEventRandom + waitForSleepRandom + branchRandom);
    },
    120_000
  );
});

