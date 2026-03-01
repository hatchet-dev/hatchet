import sleep from '@hatchet/util/sleep';
import { V1TaskStatus } from '@hatchet/clients/rest/generated/data-contracts';
import { makeE2EClient, poll } from '../__e2e__/harness';
import { concurrencyCancelNewestWorkflow } from './workflow';

describe('concurrency-cancel-newest-e2e', () => {
  const hatchet = makeE2EClient();

  it('cancels newest runs when concurrency limit reached', async () => {
    const testRunId = crypto.randomUUID();

    const toRun = await concurrencyCancelNewestWorkflow.runNoWait(
      { group: 'A' },
      { additionalMetadata: { test_run_id: testRunId } }
    );

    await sleep(1000);

    const toCancel = await Promise.all(
      Array.from({ length: 10 }, () =>
        concurrencyCancelNewestWorkflow.runNoWait(
          { group: 'A' },
          { additionalMetadata: { test_run_id: testRunId } }
        )
      )
    );

    await toRun.output;

    for (const ref of toCancel) {
      try {
        await ref.output;
      } catch {
        // Expected for cancelled runs
      }
    }

    const listResp = await poll(
      async () =>
        hatchet.runs.list({
          additionalMetadata: { test_run_id: testRunId },
          onlyTasks: false,
        } as any),
      {
        timeoutMs: 30_000,
        intervalMs: 200,
        label: 'runs list with terminal statuses',
        shouldStop: (r) => {
          const rows = r.rows || [];
          return (
            rows.length === 11 &&
            rows.every(
              (x: any) =>
                x.status !== V1TaskStatus.RUNNING && x.status !== V1TaskStatus.QUEUED
            )
          );
        },
      }
    );

    const details = (await hatchet.runs.get(toRun)) as { run?: { status?: string } };
    expect(details.run?.status).toBe(V1TaskStatus.COMPLETED);

    const runId = await toRun.getWorkflowRunId();
    const allRuns = listResp.rows || [];

    const otherRuns = allRuns.filter((r: any) => r.metadata?.id !== runId);
    expect(otherRuns.every((r: any) => r.status === V1TaskStatus.CANCELLED)).toBe(true);
  }, 90_000);
});
