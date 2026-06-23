import { randomUUID } from 'crypto';
import { makeE2EClient, poll } from '../__e2e__/harness';
import { runDetailTestWorkflow } from './workflow';
import { V1TaskStatus } from '../../../clients/rest/generated/data-contracts';

describe('getDetails-e2e', () => {
  const hatchet = makeE2EClient();

  it('returns gRPC run details that eventually show the run as done', async () => {
    const mockInput = { foo: randomUUID() };

    const ref = await runDetailTestWorkflow.runNoWait(mockInput);

    const details = await poll(async () => hatchet.runs.getDetails(ref), {
      timeoutMs: 60_000,
      intervalMs: 500,
      label: 'getDetails done=true',
      shouldStop: (d) => d.done,
    });
    console.info(details);
    expect(details.done).toBe(true);
    expect([V1TaskStatus.COMPLETED, V1TaskStatus.FAILED, V1TaskStatus.CANCELLED]).toContain(
      details.status
    );
    expect(Object.keys(details.taskRuns).length).toBeGreaterThan(0);

    for (const taskRun of Object.values(details.taskRuns)) {
      expect(taskRun.externalId).toBeTruthy();
      expect(taskRun.readableId).toBeTruthy();
      expect([V1TaskStatus.COMPLETED, V1TaskStatus.FAILED, V1TaskStatus.CANCELLED]).toContain(
        taskRun.status
      );
    }
  }, 120_000);

  it('can be called mid-execution and returns in-progress task runs', async () => {
    const ref = await runDetailTestWorkflow.runNoWait({ foo: randomUUID() });

    // Poll until at least one task run appears (run has started)
    const details = await poll(async () => hatchet.runs.getDetails(ref), {
      timeoutMs: 30_000,
      intervalMs: 100,
      label: 'getDetails has task runs',
      shouldStop: (d) => Object.keys(d.taskRuns).length > 0,
    });

    expect(details.status).toBeDefined();
    expect(Object.keys(details.taskRuns).length).toBeGreaterThan(0);
  }, 60_000);
});
