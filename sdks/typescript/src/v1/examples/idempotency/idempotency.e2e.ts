import sleep from '@hatchet/util/sleep';
import { randomUUID } from 'crypto';
import { V1TaskStatus } from '@hatchet/clients/rest/generated/data-contracts';
import { makeE2EClient, poll, startWorker, stopWorker } from '../__e2e__/harness';
import { EVENT_KEY, idempotentTask, idempotentTaskShortWindow } from './workflow';

describe('idempotency-e2e', () => {
  const hatchet = makeE2EClient();
  let worker: Awaited<ReturnType<typeof startWorker>> | undefined;

  beforeAll(async () => {
    worker = await startWorker({
      client: hatchet,
      name: 'idempotency-e2e-worker',
      workflows: [idempotentTask, idempotentTaskShortWindow],
    });
  });

  afterAll(async () => {
    await stopWorker(worker);
  });

  async function waitForRuns(testRunId: string, minRuns: number) {
    return poll(
      async () =>
        hatchet.runs.list({
          additionalMetadata: { test_run_id: testRunId },
          onlyTasks: false,
          limit: 20,
        }),
      {
        timeoutMs: 30_000,
        intervalMs: 200,
        label: `runs for ${testRunId}`,
        shouldStop: (response) => (response.rows || []).length >= minRuns,
      }
    );
  }

  it('prevents duplicate direct triggers', async () => {
    const testRunId = randomUUID();
    const ref1 = await idempotentTask.runNoWait(
      { id: testRunId },
      {
        additionalMetadata: {
          test_run_id: testRunId,
        },
      }
    );

    let collided = false;

    try {
      await idempotentTask.runNoWait({ id: testRunId });
    } catch {
      collided = true;
    }

    expect(collided).toBe(true);

    const runs = await waitForRuns(testRunId, 1);

    expect(runs.rows).toHaveLength(1);
    expect(runs.rows?.[0]?.metadata.id).toBe(await ref1.getWorkflowRunId());
  }, 60_000);

  it('allows reruns after the short idempotency window expires', async () => {
    const testRunId = randomUUID();

    for (let i = 0; i < 4; i += 1) {
      try {
        await idempotentTaskShortWindow.runNoWait(
          { id: testRunId },
          {
            additionalMetadata: {
              test_run_id: testRunId,
            },
          }
        );
      } catch {}

      if (i !== 3) {
        await sleep((i + 1.5) * 1000);
      }
    }

    const runs = await waitForRuns(testRunId, 3);

    expect(runs.rows).toHaveLength(3);
  }, 60_000);

  it('deduplicates event-triggered runs', async () => {
    const testRunId = randomUUID();
    const e1 = await hatchet.events.push(
      EVENT_KEY,
      { id: testRunId },
      {
        additionalMetadata: {
          test_run_id: testRunId,
        },
      }
    );
    const e2 = await hatchet.events.push(
      EVENT_KEY,
      { id: testRunId },
      {
        additionalMetadata: {
          test_run_id: testRunId,
        },
      }
    );

    const runs = await waitForRuns(testRunId, 1);

    expect(runs.rows).toHaveLength(1);

    const details = await poll(
      async () => hatchet.events.list({ eventIds: [e1.eventId, e2.eventId] }),
      {
        timeoutMs: 30_000,
        intervalMs: 200,
        label: 'idempotency event details',
        shouldStop: (response) => (response.rows || []).length === 2,
      }
    );

    const allTriggeredRuns = (details.rows || []).flatMap((row) => row.triggeredRuns || []);

    expect(allTriggeredRuns).toHaveLength(1);

    const runDetails = await poll(
      async () => hatchet.runs.get(allTriggeredRuns[0].workflowRunId),
      {
        timeoutMs: 30_000,
        intervalMs: 200,
        label: 'idempotent event-triggered run completion',
        shouldStop: (response) => {
          const status = response.run?.status;
          return status !== V1TaskStatus.QUEUED && status !== V1TaskStatus.RUNNING;
        },
      }
    );

    expect(runDetails.run?.status).toBe(V1TaskStatus.COMPLETED);
  }, 60_000);
});